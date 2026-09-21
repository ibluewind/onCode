package com.oncode.server.agentruntime;

import com.oncode.server.inferencegateway.ModelProfile;
import java.util.List;

/**
 * 한 에이전트의 경계. 허용 도구와 프로파일을 고정한다.
 *
 * @param id 에이전트
 * @param modelProfile Inference Gateway 프로파일. 모델 ID 아님
 * @param allowedTools 호출 가능한 Local Agent 도구. apply는 없어야 한다
 * @param outputSchemaName 출력 JSON 스키마 이름
 */
public record AgentDefinition(
    AgentId id, ModelProfile modelProfile, List<String> allowedTools, String outputSchemaName) {}
