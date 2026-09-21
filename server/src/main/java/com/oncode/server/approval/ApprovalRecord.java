package com.oncode.server.approval;

import java.time.Instant;

/**
 * {@code approval} 행. 워크플로 상태와 별도로 한 리소스에 묶인 승인이다.
 *
 * @param approvalId APR-...
 * @param workItemId 소속 워크 아이템
 * @param workflowId 소속 워크플로
 * @param approvalType DESIGN 또는 CODE_CHANGE
 * @param resourceType DESIGN_REF 또는 CHANGE_SET
 * @param resourceId 바인딩된 리소스
 * @param status 수명 상태
 * @param requestedAt 요청 시각 (UTC)
 * @param respondedAt 결정 시각. 대기면 null
 * @param approvedBy 결정 주체. 대기면 null
 * @param decision APPROVE/REJECT. 대기면 null
 * @param reason 사유. 없으면 null
 * @param expiresAt 만료 시각. 없으면 null
 * @param consumed 사용 여부
 * @param consumedAt 사용 시각. 아니면 null
 */
public record ApprovalRecord(
    String approvalId,
    String workItemId,
    String workflowId,
    ApprovalType approvalType,
    String resourceType,
    String resourceId,
    ApprovalStatus status,
    Instant requestedAt,
    Instant respondedAt,
    String approvedBy,
    ApprovalDecision decision,
    String reason,
    Instant expiresAt,
    boolean consumed,
    Instant consumedAt) {}
