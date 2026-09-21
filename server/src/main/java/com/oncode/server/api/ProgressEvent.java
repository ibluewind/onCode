package com.oncode.server.api;

/**
 * 워크플로 진행 한 단계. LLM reasoning이 아니다.
 *
 * @param stage WorkflowState 이름
 * @param status RUNNING / WAITING / FAILED
 * @param message 사용자용 짧은 설명
 */
public record ProgressEvent(String stage, String status, String message) {}
