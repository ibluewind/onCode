package idebridge

import (
	"fmt"
	"net"
	"strings"
)

// ValidateBind은 IDE IPC listen 주소가 루프백 IP인지 확인한다.
// host는 호스트명 없이 IP만 허용한다. 빈 값, 0.0.0.0, ::, 사설/공인 주소는 오류다.
func ValidateBind(host string) error {
	host = strings.TrimSpace(host)
	if host == "" {
		return fmt.Errorf("ide ipc bind must be a loopback IP, got empty")
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return fmt.Errorf("ide ipc bind must be a loopback IP, got %q", host)
	}
	if !ip.IsLoopback() {
		return fmt.Errorf("ide ipc bind must be loopback, got %q", host)
	}
	return nil
}
