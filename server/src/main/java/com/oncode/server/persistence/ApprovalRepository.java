package com.oncode.server.persistence;

import com.oncode.server.approval.ApprovalDecision;
import com.oncode.server.approval.ApprovalRecord;
import com.oncode.server.approval.ApprovalStatus;
import com.oncode.server.approval.ApprovalType;
import java.sql.Timestamp;
import java.time.Instant;
import java.util.Optional;
import org.springframework.jdbc.core.simple.JdbcClient;
import org.springframework.stereotype.Repository;

/**
 * {@code approval} 삽입·조회·소비. 워크플로 전이는 하지 않는다.
 */
@Repository
public class ApprovalRepository {

  private final JdbcClient jdbc;

  /**
   * JDBC 클라이언트를 받는다.
   *
   * @param jdbc 기본 DataSource. null이면 안 된다
   */
  public ApprovalRepository(JdbcClient jdbc) {
    this.jdbc = jdbc;
  }

  /**
   * 새 승인 요청 행을 넣는다.
   *
   * @param record {@code REQUESTED} 행. 식별자는 필수
   */
  public void insert(ApprovalRecord record) {
    jdbc.sql(
            """
            INSERT INTO oncode.approval (
              approval_id, work_item_id, workflow_id, approval_type, resource_type, resource_id,
              status, requested_at, responded_at, approved_by, decision, reason, expires_at,
              is_consumed, consumed_at
            ) VALUES (
              :id, :workItemId, :workflowId, :type, :resourceType, :resourceId,
              :status, :requested, :responded, :approvedBy, :decision, :reason, :expires,
              :consumed, :consumedAt
            )
            """)
        .param("id", record.approvalId())
        .param("workItemId", record.workItemId())
        .param("workflowId", record.workflowId())
        .param("type", record.approvalType().name())
        .param("resourceType", record.resourceType())
        .param("resourceId", record.resourceId())
        .param("status", record.status().name())
        .param("requested", WorkItemRepository.ts(record.requestedAt()))
        .param("responded", WorkItemRepository.ts(record.respondedAt()))
        .param("approvedBy", record.approvedBy())
        .param("decision", record.decision() == null ? null : record.decision().name())
        .param("reason", record.reason())
        .param("expires", WorkItemRepository.ts(record.expiresAt()))
        .param("consumed", record.consumed())
        .param("consumedAt", WorkItemRepository.ts(record.consumedAt()))
        .update();
  }

  /**
   * 승인 한 행을 읽는다.
   *
   * @param approvalId APR-...
   * @return 없으면 empty
   */
  public Optional<ApprovalRecord> findById(String approvalId) {
    return jdbc.sql(
            """
            SELECT approval_id, work_item_id, workflow_id, approval_type, resource_type, resource_id,
                   status, requested_at, responded_at, approved_by, decision, reason, expires_at,
                   is_consumed, consumed_at
            FROM oncode.approval
            WHERE approval_id = :id
            """)
        .param("id", approvalId)
        .query((rs, rowNum) -> mapRow(rs))
        .optional();
  }

  /**
   * 워크플로의 대기 중 승인을 종류별로 읽는다.
   *
   * @param workflowId WF-...
   * @param type DESIGN 또는 CODE_CHANGE
   * @return 없으면 empty
   */
  public Optional<ApprovalRecord> findRequested(String workflowId, ApprovalType type) {
    return jdbc.sql(
            """
            SELECT approval_id, work_item_id, workflow_id, approval_type, resource_type, resource_id,
                   status, requested_at, responded_at, approved_by, decision, reason, expires_at,
                   is_consumed, consumed_at
            FROM oncode.approval
            WHERE workflow_id = :workflowId
              AND approval_type = :type
              AND status = 'REQUESTED'
            """)
        .param("workflowId", workflowId)
        .param("type", type.name())
        .query((rs, rowNum) -> mapRow(rs))
        .optional();
  }

  /**
   * REQUESTED 행만 결정 결과로 갱신한다. 이미 쓰인 행은 0건.
   *
   * @param updated 소비된 행. status는 CONSUMED 또는 REJECTED
   * @return 갱신 건수 (0 또는 1)
   */
  public int consumeIfRequested(ApprovalRecord updated) {
    return jdbc.sql(
            """
            UPDATE oncode.approval
            SET status = :status,
                responded_at = :responded,
                approved_by = :approvedBy,
                decision = :decision,
                reason = :reason,
                is_consumed = :consumed,
                consumed_at = :consumedAt
            WHERE approval_id = :id AND status = 'REQUESTED'
            """)
        .param("status", updated.status().name())
        .param("responded", WorkItemRepository.ts(updated.respondedAt()))
        .param("approvedBy", updated.approvedBy())
        .param("decision", updated.decision() == null ? null : updated.decision().name())
        .param("reason", updated.reason())
        .param("consumed", updated.consumed())
        .param("consumedAt", WorkItemRepository.ts(updated.consumedAt()))
        .param("id", updated.approvalId())
        .update();
  }

  /**
   * 만료로 표시한다. 이미 결정된 행은 건드리지 않는다.
   *
   * @param approvalId APR-...
   * @param now 표시 시각 (UTC)
   * @return 갱신 건수
   */
  public int markExpired(String approvalId, Instant now) {
    return jdbc.sql(
            """
            UPDATE oncode.approval
            SET status = 'EXPIRED', responded_at = :now
            WHERE approval_id = :id AND status = 'REQUESTED'
            """)
        .param("now", Timestamp.from(now))
        .param("id", approvalId)
        .update();
  }

  private static ApprovalRecord mapRow(java.sql.ResultSet rs) throws java.sql.SQLException {
    String decision = rs.getString("decision");
    return new ApprovalRecord(
        rs.getString("approval_id"),
        rs.getString("work_item_id"),
        rs.getString("workflow_id"),
        ApprovalType.valueOf(rs.getString("approval_type")),
        rs.getString("resource_type"),
        rs.getString("resource_id"),
        ApprovalStatus.valueOf(rs.getString("status")),
        rs.getTimestamp("requested_at").toInstant(),
        tsOrNull(rs.getTimestamp("responded_at")),
        rs.getString("approved_by"),
        decision == null ? null : ApprovalDecision.valueOf(decision),
        rs.getString("reason"),
        tsOrNull(rs.getTimestamp("expires_at")),
        rs.getBoolean("is_consumed"),
        tsOrNull(rs.getTimestamp("consumed_at")));
  }

  private static Instant tsOrNull(Timestamp value) {
    return value == null ? null : value.toInstant();
  }
}
