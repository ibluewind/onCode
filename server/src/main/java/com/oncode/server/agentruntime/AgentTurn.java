package com.oncode.server.agentruntime;

import java.util.List;

/**
 * 에이전트 한 턴. 다른 에이전트 출력은 payload가 아니라 context_ref만 담는다.
 *
 * @param workItemId WI-...
 * @param workflowId WF-...
 * @param contextRefs 읽을 {@code ctx://...} 목록. payload 아님
 * @param userMessage 사용자 원문 또는 지시. 비면 안 된다
 */
public record AgentTurn(
    String workItemId, String workflowId, List<String> contextRefs, String userMessage) {}
