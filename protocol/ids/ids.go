package ids

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

type Kind string

const (
	Message   Kind = "MSG"
	Call      Kind = "CALL"
	Session   Kind = "SES"
	Project   Kind = "PRJ"
	Workspace Kind = "WS"
	WorkItem  Kind = "WI"
	Workflow  Kind = "WF"
	Approval  Kind = "APR"
	ChangeSet Kind = "CHG"
	Task      Kind = "TASK"
	Execution Kind = "EXEC"
)

var kinds = map[Kind]struct{}{
	Message: {}, Call: {}, Session: {}, Project: {}, Workspace: {},
	WorkItem: {}, Workflow: {}, Approval: {}, ChangeSet: {}, Task: {}, Execution: {},
}

func Known(k Kind) bool {
	_, ok := kinds[k]
	return ok
}

func Prefixes() []Kind {
	return []Kind{
		Message, Call, Session, Project, Workspace,
		WorkItem, Workflow, Approval, ChangeSet, Task, Execution,
	}
}

func New(kind Kind) (string, error) {
	if !Known(kind) {
		return "", fmt.Errorf("unknown id kind %q", kind)
	}
	u, err := NewV7()
	if err != nil {
		return "", err
	}
	return string(kind) + "-" + FormatUUID(u), nil
}

func MustNew(kind Kind) string {
	id, err := New(kind)
	if err != nil {
		panic(err)
	}
	return id
}

func Parse(id string) (Kind, [16]byte, error) {
	kind, uuidHex, err := split(id)
	if err != nil {
		return "", [16]byte{}, err
	}
	if !Known(kind) {
		return "", [16]byte{}, fmt.Errorf("unknown id kind in %q", id)
	}
	raw, err := parseUUID(uuidHex)
	if err != nil {
		return "", [16]byte{}, fmt.Errorf("id %q: %w", id, err)
	}
	if version(raw) != 7 {
		return "", [16]byte{}, fmt.Errorf("id %q: uuid version must be 7", id)
	}
	if variant(raw) != 2 {
		return "", [16]byte{}, fmt.Errorf("id %q: uuid variant must be RFC 4122", id)
	}
	return kind, raw, nil
}

func Validate(kind Kind, id string) error {
	got, _, err := Parse(id)
	if err != nil {
		return err
	}
	if got != kind {
		return fmt.Errorf("id %q: want kind %s", id, kind)
	}
	return nil
}

func split(id string) (Kind, string, error) {
	id = strings.TrimSpace(id)
	i := strings.IndexByte(id, '-')
	if i <= 0 || i == len(id)-1 {
		return "", "", fmt.Errorf("invalid id %q", id)
	}
	return Kind(id[:i]), id[i+1:], nil
}

func NewV7() ([16]byte, error) {
	var u [16]byte
	if _, err := rand.Read(u[:]); err != nil {
		return [16]byte{}, err
	}
	ms := uint64(time.Now().UnixMilli())
	u[0] = byte(ms >> 40)
	u[1] = byte(ms >> 32)
	u[2] = byte(ms >> 24)
	u[3] = byte(ms >> 16)
	u[4] = byte(ms >> 8)
	u[5] = byte(ms)
	u[6] = (u[6] & 0x0f) | 0x70
	u[8] = (u[8] & 0x3f) | 0x80
	return u, nil
}

func FormatUUID(u [16]byte) string {
	b := make([]byte, 36)
	hex.Encode(b[0:8], u[0:4])
	b[8] = '-'
	hex.Encode(b[9:13], u[4:6])
	b[13] = '-'
	hex.Encode(b[14:18], u[6:8])
	b[18] = '-'
	hex.Encode(b[19:23], u[8:10])
	b[23] = '-'
	hex.Encode(b[24:36], u[10:16])
	return string(b)
}

func parseUUID(s string) ([16]byte, error) {
	var u [16]byte
	if len(s) != 36 || s[8] != '-' || s[13] != '-' || s[18] != '-' || s[23] != '-' {
		return u, fmt.Errorf("invalid uuid format")
	}
	compact := s[0:8] + s[9:13] + s[14:18] + s[19:23] + s[24:36]
	n, err := hex.Decode(u[:], []byte(compact))
	if err != nil || n != 16 {
		return u, fmt.Errorf("invalid uuid hex")
	}
	return u, nil
}

func version(u [16]byte) int {
	return int(u[6] >> 4)
}

func variant(u [16]byte) int {
	return int(u[8] >> 6)
}
