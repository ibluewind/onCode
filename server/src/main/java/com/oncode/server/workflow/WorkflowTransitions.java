package com.oncode.server.workflow;

import java.util.Map;
import java.util.Optional;
import java.util.Set;

/**
 * 허용된 공식 전이만 보관한다. LLM 제안은 여기에 없다.
 * 선형 스텁은 WAITING_DESIGN_APPROVAL과 WAITING_CODE_APPROVAL에서 끊긴다.
 * 대기 탈출은 승인 결정만 허용한다.
 */
public final class WorkflowTransitions {

  private static final Map<WorkflowState, Set<WorkflowState>> ALLOWED =
      Map.of(
          WorkflowState.RECEIVED,
          Set.of(WorkflowState.CLASSIFYING),
          WorkflowState.CLASSIFYING,
          Set.of(WorkflowState.DISCOVERING_CONTEXT),
          WorkflowState.DISCOVERING_CONTEXT,
          Set.of(WorkflowState.DESIGNING),
          WorkflowState.DESIGNING,
          Set.of(WorkflowState.WAITING_DESIGN_APPROVAL),
          WorkflowState.WAITING_DESIGN_APPROVAL,
          Set.of(WorkflowState.IMPLEMENTING, WorkflowState.REJECTED),
          WorkflowState.IMPLEMENTING,
          Set.of(WorkflowState.REVIEWING),
          WorkflowState.REVIEWING,
          Set.of(WorkflowState.PREPARING_CHANGE),
          WorkflowState.PREPARING_CHANGE,
          Set.of(WorkflowState.WAITING_CODE_APPROVAL),
          WorkflowState.WAITING_CODE_APPROVAL,
          Set.of(WorkflowState.APPLYING_CHANGE, WorkflowState.REJECTED),
          WorkflowState.APPLYING_CHANGE,
          Set.of(WorkflowState.COMPLETED));

  private WorkflowTransitions() {}

  /**
   * {@code from}에서 {@code to}로 가는 전이가 허용되는지 본다.
   * 최초 RECEIVED 진입은 {@code from == null}만 허용한다.
   *
   * @param from 출발 상태. 최초 생성이면 null
   * @param to 도착 상태. null이면 거부
   * @return 허용되면 true
   */
  public static boolean allows(WorkflowState from, WorkflowState to) {
    if (to == null) {
      return false;
    }
    if (from == null) {
      return to == WorkflowState.RECEIVED;
    }
    Set<WorkflowState> next = ALLOWED.get(from);
    return next != null && next.contains(to);
  }

  /**
   * 에이전트 없이 돌리는 선형 경로의 다음 상태를 돌려준다.
   * 승인 대기와 끝 상태는 empty다.
   *
   * @param from 현재 상태. null 불가
   * @return 다음 상태. 대기·끝이면 empty
   */
  public static Optional<WorkflowState> nextLinear(WorkflowState from) {
    if (from == null || from.isWaiting() || from.isTerminal()) {
      return Optional.empty();
    }
    // apply 완료는 Local Agent 통지 후에만. 선형 스텁이 COMPLETED로 뛰지 않는다.
    if (from == WorkflowState.APPLYING_CHANGE) {
      return Optional.empty();
    }
    Set<WorkflowState> next = ALLOWED.get(from);
    if (next == null || next.isEmpty()) {
      return Optional.empty();
    }
    return Optional.of(next.iterator().next());
  }
}
