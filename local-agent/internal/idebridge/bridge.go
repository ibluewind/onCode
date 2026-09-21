package idebridge

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	ipcPath         = "/ipc"
	maxMessageBytes = 64 << 10
	registerTimeout = 5 * time.Second
	shutdownTimeout = 2 * time.Second
	readIdleTimeout = 60 * time.Second
)

// Options는 IDE Bridge listen 설정이다. Bind는 루프백 IP여야 한다.
type Options struct {
	Bind     string
	Port     int
	StateDir string
	AgentID  string
	// Handler는 ide.register 이후 프레임을 처리한다. nil이면 메시지를 버린다.
	Handler Handler
}

// Handler는 등록된 IDE 클라이언트의 JSON 프레임을 처리한다.
// 동기 응답이 있으면 반환하고, 비동기(progress)는 Bridge.Broadcast로 보낸다.
type Handler func(clientID string, msg Message) *Message

type clientConn struct {
	id      string
	conn    *websocket.Conn
	writeMu sync.Mutex
}

func (c *clientConn) write(msg Message) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return c.conn.WriteJSON(msg)
}

// Bridge는 loopback WebSocket을 열고 세션 토큰을 state dir에 쓴다.
// 연결마다 ide.register를 받으며, 워크스페이스 도구는 호출하지 않는다.
type Bridge struct {
	bind     string
	port     int
	stateDir string
	agentID  string
	token    string
	handler  Handler

	mu       sync.Mutex
	started  bool
	endpoint Endpoint
	httpSrv  *http.Server
	listener net.Listener
	conns    map[*websocket.Conn]*clientConn
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		return originAllowed(r.Header.Get("Origin"))
	},
}

// New는 토큰을 발급하고 Bridge를 구성한다. Listen은 하지 않는다.
// Bind가 루프백이 아니면 오류다. 재시작마다 New를 호출하면 토큰이 회전한다.
func New(opts Options) (*Bridge, error) {
	if err := ValidateBind(opts.Bind); err != nil {
		return nil, err
	}
	if opts.Port < 0 || opts.Port > 65535 {
		return nil, fmt.Errorf("ide ipc port out of range: %d", opts.Port)
	}
	if opts.StateDir == "" {
		return nil, fmt.Errorf("ide ipc state dir is required")
	}
	if opts.AgentID == "" {
		return nil, fmt.Errorf("ide ipc agent id is required")
	}
	token, err := newToken()
	if err != nil {
		return nil, err
	}
	return &Bridge{
		bind:     opts.Bind,
		port:     opts.Port,
		stateDir: opts.StateDir,
		agentID:  opts.AgentID,
		token:    token,
		handler:  opts.Handler,
		conns:    make(map[*websocket.Conn]*clientConn),
	}, nil
}

// SetHandler는 기동 후 채팅 허브를 붙일 때 쓴다. 기존 연결에는 소급 적용되지 않는다.
func (b *Bridge) SetHandler(h Handler) {
	if b == nil {
		return
	}
	b.mu.Lock()
	b.handler = h
	b.mu.Unlock()
}

// Broadcast는 연결된 모든 IDE 클라이언트에 프레임을 보낸다. 워크스페이스 I/O는 없다.
func (b *Bridge) Broadcast(msg Message) {
	if b == nil {
		return
	}
	b.mu.Lock()
	clients := make([]*clientConn, 0, len(b.conns))
	for _, c := range b.conns {
		clients = append(clients, c)
	}
	b.mu.Unlock()
	for _, c := range clients {
		_ = c.write(msg)
	}
}

// Start는 루프백에 listen하고 ide-ipc.json을 쓴 뒤 HTTP 서버를 고루틴에서 돌린다.
// ctx가 취소되면 Close와 같이 종료한다. 이미 시작된 Bridge에 다시 호출하면 오류다.
func (b *Bridge) Start(ctx context.Context) error {
	if b == nil {
		return fmt.Errorf("ide ipc bridge is nil")
	}
	b.mu.Lock()
	if b.started {
		b.mu.Unlock()
		return fmt.Errorf("ide ipc bridge already started")
	}
	b.mu.Unlock()

	addr := net.JoinHostPort(b.bind, fmt.Sprintf("%d", b.port))
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("ide ipc listen: %w", err)
	}
	tcpAddr, ok := ln.Addr().(*net.TCPAddr)
	if !ok || tcpAddr.IP == nil || !tcpAddr.IP.IsLoopback() {
		_ = ln.Close()
		return fmt.Errorf("ide ipc refused non-loopback listener %s", ln.Addr())
	}

	mux := http.NewServeMux()
	mux.HandleFunc(ipcPath, b.handleIPC)
	srv := &http.Server{Handler: mux}

	ep := Endpoint{
		Bind:  b.bind,
		Port:  tcpAddr.Port,
		Path:  ipcPath,
		Token: b.token,
	}
	if err := writeEndpoint(b.stateDir, ep); err != nil {
		_ = ln.Close()
		return err
	}

	b.mu.Lock()
	b.started = true
	b.listener = ln
	b.httpSrv = srv
	b.endpoint = ep
	b.mu.Unlock()

	go func() {
		_ = srv.Serve(ln)
	}()
	go func() {
		<-ctx.Done()
		_ = b.Close()
	}()
	return nil
}

// Endpoint는 디스커버리 파일에 쓴 현재 listen 정보다. Start 전에 호출하면 오류다.
func (b *Bridge) Endpoint() (Endpoint, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.started || b.endpoint.Port == 0 {
		return Endpoint{}, fmt.Errorf("ide ipc not listening")
	}
	return b.endpoint, nil
}

// Close는 연결을 끊고 listen을 닫으며 디스커버리 파일을 삭제한다.
// 토큰은 재사용하지 않는다. 다음 기동은 New로 새 토큰을 만든다.
func (b *Bridge) Close() error {
	if b == nil {
		return nil
	}
	b.mu.Lock()
	srv := b.httpSrv
	conns := b.conns
	b.conns = make(map[*websocket.Conn]*clientConn)
	b.httpSrv = nil
	b.listener = nil
	b.started = false
	b.mu.Unlock()

	for _, c := range conns {
		_ = c.conn.Close()
	}
	var err error
	if srv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		err = srv.Shutdown(ctx)
		cancel()
	}
	removeEndpoint(b.stateDir)
	return err
}

// handleIPC는 토큰·Origin을 검사한 뒤에만 WebSocket으로 올린다.
// 실패 시 업그레이드하지 않고 401/403을 돌려 fail-closed 한다.
func (b *Bridge) handleIPC(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !originAllowed(r.Header.Get("Origin")) {
		http.Error(w, "forbidden origin", http.StatusForbidden)
		return
	}
	if !b.tokenMatches(extractToken(r)) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	b.serveConn(conn)
}

// tokenMatches는 제공된 토큰이 현재 세션 토큰과 같은지 상수시간 비교한다.
func (b *Bridge) tokenMatches(got string) bool {
	if got == "" || b.token == "" || len(got) != len(b.token) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(b.token)) == 1
}

// extractToken은 query token 또는 Authorization Bearer를 읽는다. 둘 다 없으면 빈 문자열이다.
func extractToken(r *http.Request) string {
	if t := strings.TrimSpace(r.URL.Query().Get("token")); t != "" {
		return t
	}
	auth := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if strings.HasPrefix(auth, prefix) {
		return strings.TrimSpace(strings.TrimPrefix(auth, prefix))
	}
	return ""
}

// serveConn은 첫 메시지로 ide.register를 강제하고 client id를 돌려준 뒤, 연결이 닫힐 때까지 읽는다.
func (b *Bridge) serveConn(conn *websocket.Conn) {
	defer conn.Close()
	conn.SetReadLimit(maxMessageBytes)
	_ = conn.SetReadDeadline(time.Now().Add(registerTimeout))

	var msg Message
	if err := conn.ReadJSON(&msg); err != nil {
		_ = conn.WriteJSON(errorMessage("", codeInvalid, "expected ide.register"))
		return
	}
	if msg.Type != typeRegister {
		_ = conn.WriteJSON(errorMessage(msg.MessageID, codeUnsupported, "first message must be ide.register"))
		return
	}
	ide, _ := msg.Data["ide"].(string)
	if strings.TrimSpace(ide) == "" {
		_ = conn.WriteJSON(errorMessage(msg.MessageID, codeInvalid, "data.ide is required"))
		return
	}

	clientID, err := newIDEClientID()
	if err != nil {
		_ = conn.WriteJSON(errorMessage(msg.MessageID, codeInvalid, "failed to allocate ide_client_id"))
		return
	}
	_ = conn.SetReadDeadline(time.Time{})
	slot := &clientConn{id: clientID, conn: conn}
	b.trackClient(slot)
	defer b.untrack(conn)
	if err := slot.write(Message{
		Type:      typeRegistered,
		MessageID: msg.MessageID,
		Data: map[string]any{
			"ide_client_id": clientID,
			"agent_id":      b.agentID,
		},
	}); err != nil {
		return
	}

	for {
		_ = conn.SetReadDeadline(time.Now().Add(readIdleTimeout))
		var next Message
		if err := conn.ReadJSON(&next); err != nil {
			return
		}
		b.mu.Lock()
		h := b.handler
		b.mu.Unlock()
		if h == nil {
			continue
		}
		if reply := h(clientID, next); reply != nil {
			_ = slot.write(*reply)
		}
	}
}

// trackClient는 브로드캐스트 대상에 연결을 넣는다.
func (b *Bridge) trackClient(c *clientConn) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.conns == nil {
		b.conns = make(map[*websocket.Conn]*clientConn)
	}
	b.conns[c.conn] = c
}

// untrack은 종료된 소켓을 맵에서 뺀다.
func (b *Bridge) untrack(conn *websocket.Conn) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.conns, conn)
}

// errorMessage는 IDE에 돌려줄 오류 프레임을 만든다.
func errorMessage(messageID, code, message string) Message {
	return Message{
		Type:      typeError,
		MessageID: messageID,
		Data: map[string]any{
			"code":    code,
			"message": message,
		},
	}
}

// newIDEClientID는 연결마다 새 ide_client_id를 만든다. protocol ID 종류를 추가하지 않는다.
func newIDEClientID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return "ide-" + hex.EncodeToString(b[:]), nil
}
