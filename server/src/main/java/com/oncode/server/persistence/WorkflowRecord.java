package com.oncode.server.persistence;

import com.oncode.server.workflow.WorkflowState;
import com.oncode.server.workflow.WorkflowStatus;
import java.time.Instant;

/**
 * {@code workflow} 행. 공식 상태와 리스의 SSOT다.
 *
 * @param workflowId 식별자 (WF-...)
 * @param workItemId 부모 워크 아이템
 * @param workflowType 워크플로 유형
 * @param currentState 현재 공식 상태
 * @param previousState 직전 상태. 최초면 null
 * @param resumeState 재개 참고 상태. 없으면 null
 * @param revision 낙관적 잠금 버전. 0부터
 * @param status 실행 수명
 * @param retryCount 재시도 횟수. 0부터
 * @param leaseOwner 리스 소유 노드. WAITING이면 null
 * @param leaseUntil 리스 만료 (UTC). 없으면 null
 * @param createdAt 생성 시각 (UTC)
 * @param updatedAt 변경 시각 (UTC)
 */
public record WorkflowRecord(
    String workflowId,
    String workItemId,
    String workflowType,
    WorkflowState currentState,
    WorkflowState previousState,
    WorkflowState resumeState,
    int revision,
    WorkflowStatus status,
    int retryCount,
    String leaseOwner,
    Instant leaseUntil,
    Instant createdAt,
    Instant updatedAt) {}
