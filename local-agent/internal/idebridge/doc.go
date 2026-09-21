// Package idebridge는 IDE ↔ Local Agent localhost WebSocket+JSON IPC다 (ADR-004).
//
// 루프백에만 바인딩하고, 기동 시 발급한 세션 토큰이 있는 연결만 업그레이드한다.
// 워크스페이스 파일 I/O, 빌드, 테스트, 서버 gRPC(ADR-003)는 다루지 않는다.
package idebridge
