package com.oncode.server.workflow;

/**
 * 워크플로 행의 실행 수명 상태. {@link WorkflowState}와 다르다.
 * WAITING_* 공식 상태에서는 이 값이 {@link #WAITING}이어야 한다.
 */
public enum WorkflowStatus {
  RUNNING,
  WAITING,
  COMPLETED,
  FAILED
}
