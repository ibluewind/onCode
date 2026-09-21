package com.oncode.server.inferencegateway;

/**
 * 모델 호출의 유일한 진입점 (ADR-007). 에이전트는 vLLM을 직접 부르지 않는다.
 * Phase 2는 실모델 없이 스크립트 구현을 붙일 수 있다.
 */
public interface InferenceGateway {

  /**
   * 프로파일로 한 번 완료한다. 모델 ID를 받지 않는다.
   *
   * @param request 프로파일과 프롬프트. null 불가, userPrompt 공백 불가
   * @return 구조화 JSON 출력
   */
  InferenceResponse complete(InferenceRequest request);
}
