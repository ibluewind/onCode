package com.oncode.server.toolgateway;

/**
 * SPEC-01 ToolError. 게이트웨이가 구조화해서 돌려준다.
 *
 * @param code PROTOCOL/TOOL/TIMEOUT 등 코드
 * @param category 분류
 * @param message 사람 읽기용
 * @param retryable 재시도 가능 여부
 */
public record ToolError(String code, String category, String message, boolean retryable) {}
