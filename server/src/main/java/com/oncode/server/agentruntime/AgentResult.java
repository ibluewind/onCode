package com.oncode.server.agentruntime;

import com.oncode.server.inferencegateway.ModelProfile;

/**
 * 에이전트 턴 결과. 본문은 Context Storage에 있고 여기엔 ref만 있다.
 *
 * @param agentId 실행한 에이전트
 * @param contextRef {@code ctx://...}
 * @param modelProfile 사용한 프로파일
 */
public record AgentResult(AgentId agentId, String contextRef, ModelProfile modelProfile) {}
