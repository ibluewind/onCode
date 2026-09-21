package com.oncode.server.toolgateway;

/**
 * 실제 Local Agent(또는 Phase 2 하니스)로 ToolCall을 전달한다.
 * 게이트웨이의 정책(apply 거부, 중복 call_id, timeout)은 여기 밖에 있다.
 */
public interface LocalAgentChannel {

  /**
   * 한 번 호출한다. 막히면 게이트웨이가 timeout으로 자른다.
   *
   * @param call 이미 call_id가 채워진 호출
   * @return Local Agent ToolResponse에 해당하는 결과
   */
  ToolResponse invoke(ToolCall call);
}
