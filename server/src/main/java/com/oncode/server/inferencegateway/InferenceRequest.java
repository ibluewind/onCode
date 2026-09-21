package com.oncode.server.inferencegateway;

/**
 * Inference Gateway로 보내는 한 번의 완료 요청.
 * 모델 ID는 두지 않는다.
 *
 * @param modelProfile 라우팅 키. null 불가
 * @param systemPrompt 시스템 프롬프트. 없으면 빈 문자열
 * @param userPrompt 사용자/컨텍스트 프롬프트. 비면 안 된다
 * @param schemaName 기대 출력 스키마 이름. 없으면 null
 */
public record InferenceRequest(
    ModelProfile modelProfile, String systemPrompt, String userPrompt, String schemaName) {}
