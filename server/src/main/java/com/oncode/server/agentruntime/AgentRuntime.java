package com.oncode.server.agentruntime;

import java.util.List;

/**
 * 공유 런타임. 모델은 Inference Gateway만, 워크스페이스는 Tool Gateway만 쓴다.
 * apply는 호출하지 않는다.
 */
public interface AgentRuntime {

  /**
   * 한 에이전트 턴을 실행하고 결과를 context_ref로 남긴다.
   *
   * @param agentId Senior / Implementation / Review
   * @param turn context_ref와 사용자 메시지. payload 필드 없음
   * @return 저장된 context_ref
   */
  AgentResult run(AgentId agentId, AgentTurn turn);

  /**
   * 등록된 경계 에이전트 목록.
   *
   * @return 3개
   */
  List<AgentDefinition> definitions();
}
