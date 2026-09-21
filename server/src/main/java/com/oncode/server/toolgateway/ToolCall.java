package com.oncode.server.toolgateway;

import java.util.Map;

/**
 * Local Agent로 보낼 도구 호출. 게이트웨이가 ToolRequest payload로 매핑한다.
 *
 * @param callId CALL-... . 비면 게이트웨이가 발급
 * @param tool Phase 1 도구명
 * @param arguments JSON object에 해당하는 맵. null이면 빈 맵
 * @param timeoutMs 제한(ms). null이면 게이트웨이 기본값
 * @param actorType USER / AGENT / ORCHESTRATOR
 * @param actorId 호출 주체 ID
 * @param workflowId WF-... . 없으면 null
 * @param workItemId WI-... . 없으면 null
 */
public record ToolCall(
    String callId,
    String tool,
    Map<String, Object> arguments,
    Integer timeoutMs,
    String actorType,
    String actorId,
    String workflowId,
    String workItemId) {}
