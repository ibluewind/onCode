package idebridge

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// EndpointFileName은 Local Agent state dir에 쓰는 IDE IPC 디스커버리 파일이다.
const EndpointFileName = "ide-ipc.json"

// Endpoint는 확장이 WebSocket에 붙는 데 필요한 연결 정보다.
// Token은 에이전트 재시작마다 새로 쓰이며(회전), 이 파일이 전달 수단이다.
type Endpoint struct {
	Bind  string `json:"bind"`
	Port  int    `json:"port"`
	Path  string `json:"path"`
	Token string `json:"token"`
}

// newToken은 32바이트 암호학 난수를 hex로 인코딩한 세션 토큰을 만든다.
func newToken() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("ide ipc token: %w", err)
	}
	return hex.EncodeToString(b[:]), nil
}

// endpointPath는 stateDir 아래 디스커버리 파일의 절대 경로를 반환한다.
func endpointPath(stateDir string) string {
	return filepath.Join(stateDir, EndpointFileName)
}

// writeEndpoint는 토큰이 포함된 endpoint를 0600 파일로 덮어쓴다.
// 실패하면 확장이 이전 토큰을 읽지 못하도록 기존 파일을 지운다.
func writeEndpoint(stateDir string, ep Endpoint) error {
	if stateDir == "" {
		return fmt.Errorf("ide ipc state dir is empty")
	}
	path := endpointPath(stateDir)
	raw, err := json.MarshalIndent(ep, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		_ = os.Remove(path)
		return fmt.Errorf("write ide ipc endpoint: %w", err)
	}
	return nil
}

// removeEndpoint는 에이전트 종료 시 디스커버리 파일을 지워 죽은 포트로 붙지 않게 한다.
func removeEndpoint(stateDir string) {
	if stateDir == "" {
		return
	}
	_ = os.Remove(endpointPath(stateDir))
}

// ReadEndpoint는 stateDir의 ide-ipc.json을 읽는다.
// 파일이 없거나 JSON이 아니면 오류다. 호출자가 토큰을 로그에 남기면 안 된다.
func ReadEndpoint(stateDir string) (Endpoint, error) {
	var ep Endpoint
	raw, err := os.ReadFile(endpointPath(stateDir))
	if err != nil {
		return Endpoint{}, err
	}
	if err := json.Unmarshal(raw, &ep); err != nil {
		return Endpoint{}, fmt.Errorf("parse ide ipc endpoint: %w", err)
	}
	if ep.Bind == "" || ep.Port <= 0 || ep.Path == "" || ep.Token == "" {
		return Endpoint{}, fmt.Errorf("incomplete ide ipc endpoint")
	}
	return ep, nil
}
