package com.oncode.server.inferencegateway;

/**
 * 게이트웨이가 돌려주는 구조화 출력. hidden reasoning은 없다.
 *
 * @param modelProfile 실제로 쓴 프로파일
 * @param outputJson 스키마에 맞는 JSON 본문
 */
public record InferenceResponse(ModelProfile modelProfile, String outputJson) {}
