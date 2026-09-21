package com.oncode.server.workflow;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;

import org.junit.jupiter.api.Test;

class WorkflowTransitionsTest {

  /**
   * 선형 경로와 승인 대기 탈출만 허용하는지 본다.
   */
  @Test
  void allowsLinearPathOnly() {
    assertThat(WorkflowTransitions.allows(null, WorkflowState.RECEIVED)).isTrue();
    assertThat(WorkflowTransitions.allows(WorkflowState.RECEIVED, WorkflowState.CLASSIFYING))
        .isTrue();
    assertThat(WorkflowTransitions.allows(WorkflowState.RECEIVED, WorkflowState.DESIGNING))
        .isFalse();
    assertThat(
            WorkflowTransitions.allows(
                WorkflowState.WAITING_DESIGN_APPROVAL, WorkflowState.DESIGNING))
        .isFalse();
    assertThat(
            WorkflowTransitions.allows(
                WorkflowState.WAITING_DESIGN_APPROVAL, WorkflowState.IMPLEMENTING))
        .isTrue();
    assertThat(
            WorkflowTransitions.allows(
                WorkflowState.WAITING_CODE_APPROVAL, WorkflowState.APPLYING_CHANGE))
        .isTrue();
    assertThat(WorkflowTransitions.allows(WorkflowState.APPLYING_CHANGE, WorkflowState.COMPLETED))
        .isTrue();
    assertThat(WorkflowTransitions.allows(WorkflowState.COMPLETED, WorkflowState.RECEIVED))
        .isFalse();
  }

  /**
   * 선형 자동 진행은 승인 대기에서 멈춘다.
   */
  @Test
  void nextLinearStopsAtWaitingStates() {
    assertThat(WorkflowTransitions.nextLinear(WorkflowState.DESIGNING))
        .contains(WorkflowState.WAITING_DESIGN_APPROVAL);
    assertThat(WorkflowTransitions.nextLinear(WorkflowState.WAITING_DESIGN_APPROVAL)).isEmpty();
    assertThat(WorkflowTransitions.nextLinear(WorkflowState.IMPLEMENTING))
        .contains(WorkflowState.REVIEWING);
    assertThat(WorkflowTransitions.nextLinear(WorkflowState.WAITING_CODE_APPROVAL)).isEmpty();
    assertThat(WorkflowTransitions.nextLinear(WorkflowState.APPLYING_CHANGE)).isEmpty();
  }
}
