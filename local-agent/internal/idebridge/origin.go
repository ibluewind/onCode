package idebridge

import (
	"net"
	"net/url"
	"strings"
)

// originAllowed는 브라우저가 아닌 로컬 IDE 클라이언트 Origin만 허용한다.
// Origin이 없거나 "null"이면 확장 호스트/테스트 클라이언트로 본다.
// http(s)는 루프백 호스트만 통과한다. 그 외 웹 Origin은 CSRF 방지를 위해 거부한다.
func originAllowed(origin string) bool {
	origin = strings.TrimSpace(origin)
	if origin == "" || origin == "null" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil || u.Scheme == "" {
		return false
	}
	switch strings.ToLower(u.Scheme) {
	case "vscode-file", "vscode-webview", "vscode-app":
		return true
	case "http", "https":
		host := u.Hostname()
		if host == "localhost" {
			return true
		}
		ip := net.ParseIP(host)
		return ip != nil && ip.IsLoopback()
	default:
		return false
	}
}
