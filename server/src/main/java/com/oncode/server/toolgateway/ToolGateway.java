package com.oncode.server.toolgateway;

/**
 * 서버가 워크스페이스에 직접 닿지 않고 Local Agent 도구만 호출한다.
 * 파일시스템 접근은 하지 않는다.
 */
public interface ToolGateway {

  /**
   * 도메인 호출을 ToolRequest로 매핑해 실행한다.
   * AGENT의 {@code workspace.apply_changes}는 PERMISSION_DENIED다.
   * 같은 call_id의 side-effect 도구는 재실행하지 않고 첫 응답을 돌려준다.
   *
   * @param call 도구 호출. tool은 비면 안 된다
   * @return 상관된 call_id를 가진 응답. timeout이면 status=TIMEOUT
   */
  ToolResponse call(ToolCall call);
}
