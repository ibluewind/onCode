package com.oncode.server.workflow;

/**
 * Implementation 워크플로의 공식 상태.
 * SPEC-02 전체 경로 중 PHASE_02가 허용한 선형 경로와 승인 대기·반려만 둔다.
 * ANALYZING_DEPENDENCY / RETRIEVING_GUIDE 등은 전이 그래프에 없다.
 */
public enum WorkflowState {
  RECEIVED,
  CLASSIFYING,
  DISCOVERING_CONTEXT,
  DESIGNING,
  WAITING_DESIGN_APPROVAL,
  IMPLEMENTING,
  REVIEWING,
  PREPARING_CHANGE,
  WAITING_CODE_APPROVAL,
  APPLYING_CHANGE,
  /** Local Agent apply 성공 후 끝. BUILDING/TESTING은 MVP에서 생략한다. */
  COMPLETED,
  REJECTED;

  /**
   * 대기 상태이면 실행 워커를 붙잡지 않는다.
   *
   * @return 이름이 {@code WAITING_}로 시작하면 true
   */
  public boolean isWaiting() {
    return name().startsWith("WAITING_");
  }

  /**
   * 더 이상 진행하지 않는 끝 상태인지 본다.
   *
   * @return COMPLETED 또는 REJECTED면 true
   */
  public boolean isTerminal() {
    return this == COMPLETED || this == REJECTED;
  }

  /**
   * 성공으로 끝난 끝 상태인지 본다.
   *
   * @return COMPLETED면 true
   */
  public boolean isCompleted() {
    return this == COMPLETED;
  }
}
