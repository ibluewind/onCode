package com.oncode.server.persistence;

import com.oncode.server.workflow.WorkflowState;
import java.time.Instant;

/**
 * {@code workflow_transition} 한 행. 이미 커밋된 전이만 담는다.
 *
 * @param transitionId SPEC-03 {@code transition_id}
 * @param workflowId 워크플로
 * @param fromState 출발. 최초 진입은 null
 * @param toState 도착
 * @param triggerType 원인 종류
 * @param triggerId 상관 ID. 없으면 null
 * @param reason 짧은 이유. 없으면 null
 * @param createdAt 전이 시각 (UTC)
 */
public record WorkflowTransitionRecord(
    String transitionId,
    String workflowId,
    WorkflowState fromState,
    WorkflowState toState,
    String triggerType,
    String triggerId,
    String reason,
    Instant createdAt) {}
