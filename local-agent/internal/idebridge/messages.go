package idebridge

// Message는 IDE Bridge JSON 프레임이다. 필드 이름은 snake_case다.
// 슬라이스 1은 ide.register / ide.registered / error만 처리한다.
type Message struct {
	Type      string         `json:"type"`
	MessageID string         `json:"message_id,omitempty"`
	Data      map[string]any `json:"data,omitempty"`
}

const (
	typeRegister   = "ide.register"
	typeRegistered = "ide.registered"
	typeError      = "error"

	codeUnauthorized = "UNAUTHORIZED"
	codeForbidden    = "FORBIDDEN"
	codeInvalid      = "INVALID_REQUEST"
	codeUnsupported  = "UNSUPPORTED"
)
