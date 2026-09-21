package com.oncode.server.approval;

import com.oncode.server.persistence.ApprovalRepository;
import com.oncode.server.persistence.EntityIds;
import java.time.Clock;
import java.time.Instant;
import java.util.Objects;
import java.util.Optional;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

/**
 * 승인 요청과 1회 소비. 워크플로 전이는 {@link com.oncode.server.workflow.WorkflowEngine}이 수행한다.
 * REST 어댑터는 이 포트 뒤에 붙인다.
 */
@Service
public class ApprovalService implements ApprovalPort {

  private final ApprovalRepository approvals;
  private final Clock clock;

  /**
   * 저장소를 받는다.
   *
   * @param approvals 승인 저장소. null 불가
   */
  @Autowired
  public ApprovalService(ApprovalRepository approvals) {
    this(approvals, Clock.systemUTC());
  }

  /**
   * 테스트에서 시계를 고정할 때 쓴다.
   *
   * @param approvals 승인 저장소
   * @param clock UTC 시계
   */
  ApprovalService(ApprovalRepository approvals, Clock clock) {
    this.approvals = approvals;
    this.clock = clock;
  }

  /**
   * {@inheritDoc}
   */
  @Override
  @Transactional
  public ApprovalRecord request(
      String workItemId,
      String workflowId,
      ApprovalType type,
      String resourceType,
      String resourceId,
      Instant expiresAt) {
    if (workItemId == null || workItemId.isBlank()) {
      throw new IllegalArgumentException("workItemId must not be blank");
    }
    if (workflowId == null || workflowId.isBlank()) {
      throw new IllegalArgumentException("workflowId must not be blank");
    }
    Objects.requireNonNull(type, "type");
    if (resourceType == null || resourceType.isBlank()) {
      throw new IllegalArgumentException("resourceType must not be blank");
    }
    if (resourceId == null || resourceId.isBlank()) {
      throw new IllegalArgumentException("resourceId must not be blank");
    }
    Instant now = clock.instant();
    ApprovalRecord created =
        new ApprovalRecord(
            EntityIds.approval(),
            workItemId,
            workflowId,
            type,
            resourceType.trim(),
            resourceId.trim(),
            ApprovalStatus.REQUESTED,
            now,
            null,
            null,
            null,
            null,
            expiresAt,
            false,
            null);
    approvals.insert(created);
    return created;
  }

  /**
   * {@inheritDoc}
   */
  @Override
  public Optional<ApprovalRecord> findById(String approvalId) {
    Objects.requireNonNull(approvalId, "approvalId");
    return approvals.findById(approvalId);
  }

  /**
   * {@inheritDoc}
   */
  @Override
  public Optional<ApprovalRecord> findRequested(String workflowId, ApprovalType type) {
    Objects.requireNonNull(workflowId, "workflowId");
    Objects.requireNonNull(type, "type");
    return approvals.findRequested(workflowId, type);
  }

  /**
   * {@inheritDoc}
   *
   * <p>조회 실패 예외는 호출자가 Rejected로 바꿀 수 있으므로 트랜잭션을 rollback-only로 표시하지 않는다.
   * 만료 마킹은 호출자 커밋에 포함된다.
   */
  @Override
  @Transactional(
      noRollbackFor = {
        ApprovalNotFoundException.class,
        ApprovalExpiredException.class,
        ApprovalReuseException.class
      })
  public ApprovalRecord requireUsable(String approvalId) {
    ApprovalRecord loaded =
        approvals.findById(approvalId).orElseThrow(() -> new ApprovalNotFoundException(approvalId));
    Instant now = clock.instant();
    if (loaded.expiresAt() != null && !loaded.expiresAt().isAfter(now)) {
      approvals.markExpired(approvalId, now);
      throw new ApprovalExpiredException(approvalId);
    }
    if (loaded.status() != ApprovalStatus.REQUESTED || loaded.consumed()) {
      throw new ApprovalReuseException(approvalId, loaded.status());
    }
    return loaded;
  }

  /**
   * {@inheritDoc}
   */
  @Override
  @Transactional
  public ApprovalRecord consume(
      String approvalId, ApprovalDecision decision, String actor, String reason) {
    Objects.requireNonNull(decision, "decision");
    ApprovalRecord pending = requireUsable(approvalId);
    Instant now = clock.instant();
    ApprovalStatus nextStatus =
        decision == ApprovalDecision.APPROVE ? ApprovalStatus.CONSUMED : ApprovalStatus.REJECTED;
    ApprovalRecord consumed =
        new ApprovalRecord(
            pending.approvalId(),
            pending.workItemId(),
            pending.workflowId(),
            pending.approvalType(),
            pending.resourceType(),
            pending.resourceId(),
            nextStatus,
            pending.requestedAt(),
            now,
            actor,
            decision,
            reason,
            pending.expiresAt(),
            true,
            now);
    int rows = approvals.consumeIfRequested(consumed);
    if (rows != 1) {
      throw new ApprovalReuseException(approvalId, pending.status());
    }
    return consumed;
  }
}
