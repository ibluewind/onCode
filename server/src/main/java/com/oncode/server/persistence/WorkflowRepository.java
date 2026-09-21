package com.oncode.server.persistence;

import com.oncode.server.workflow.WorkflowState;
import com.oncode.server.workflow.WorkflowStatus;
import java.sql.Timestamp;
import java.time.Instant;
import java.util.Optional;
import org.springframework.jdbc.core.simple.JdbcClient;
import org.springframework.stereotype.Repository;

/**
 * {@code workflow} 삽입·조회·리스·낙관적 전이. 허용 그래프는 검사하지 않는다.
 */
@Repository
public class WorkflowRepository {

  private final JdbcClient jdbc;

  /**
   * JDBC 클라이언트를 받는다.
   *
   * @param jdbc 기본 DataSource. null이면 안 된다
   */
  public WorkflowRepository(JdbcClient jdbc) {
    this.jdbc = jdbc;
  }

  /**
   * 새 워크플로 행을 삽입한다.
   *
   * @param record 저장할 행. 식별자와 상태는 필수
   */
  public void insert(WorkflowRecord record) {
    jdbc.sql(
            """
            INSERT INTO oncode.workflow (
              workflow_id, work_item_id, workflow_type, current_state, previous_state,
              resume_state, revision, status, retry_count, lease_owner, lease_until,
              created_at, updated_at
            ) VALUES (
              :id, :workItemId, :type, :current, :previous, :resume, :revision, :status,
              :retry, :leaseOwner, :leaseUntil, :created, :updated
            )
            """)
        .param("id", record.workflowId())
        .param("workItemId", record.workItemId())
        .param("type", record.workflowType())
        .param("current", record.currentState().name())
        .param("previous", nameOrNull(record.previousState()))
        .param("resume", nameOrNull(record.resumeState()))
        .param("revision", record.revision())
        .param("status", record.status().name())
        .param("retry", record.retryCount())
        .param("leaseOwner", record.leaseOwner())
        .param("leaseUntil", WorkItemRepository.ts(record.leaseUntil()))
        .param("created", WorkItemRepository.ts(record.createdAt()))
        .param("updated", WorkItemRepository.ts(record.updatedAt()))
        .update();
  }

  /**
   * 워크플로 한 행을 읽는다.
   *
   * @param workflowId WF-...
   * @return 없으면 empty
   */
  public Optional<WorkflowRecord> findById(String workflowId) {
    return jdbc.sql(
            """
            SELECT workflow_id, work_item_id, workflow_type, current_state, previous_state,
                   resume_state, revision, status, retry_count, lease_owner, lease_until,
                   created_at, updated_at
            FROM oncode.workflow
            WHERE workflow_id = :id
            """)
        .param("id", workflowId)
        .query((rs, rowNum) -> mapRow(rs))
        .optional();
  }

  /**
   * 워크 아이템에 묶인 워크플로를 읽는다. 1일차는 1:1.
   *
   * @param workItemId WI-...
   * @return 없으면 empty
   */
  public Optional<WorkflowRecord> findByWorkItemId(String workItemId) {
    return jdbc.sql(
            """
            SELECT workflow_id, work_item_id, workflow_type, current_state, previous_state,
                   resume_state, revision, status, retry_count, lease_owner, lease_until,
                   created_at, updated_at
            FROM oncode.workflow
            WHERE work_item_id = :id
            """)
        .param("id", workItemId)
        .query((rs, rowNum) -> mapRow(rs))
        .optional();
  }

  /**
   * 예상 {@code revision}과 일치할 때만 상태·리스를 갱신한다.
   *
   * @param updated 새 값. {@code revision}은 이미 +1 된 값
   * @param expectedRevision 읽었던 버전. 불일치하면 0행
   * @return 갱신된 행 수 (0 또는 1)
   */
  public int updateState(WorkflowRecord updated, int expectedRevision) {
    return jdbc.sql(
            """
            UPDATE oncode.workflow
            SET current_state = :current,
                previous_state = :previous,
                resume_state = :resume,
                revision = :revision,
                status = :status,
                lease_owner = :leaseOwner,
                lease_until = :leaseUntil,
                updated_at = :updated
            WHERE workflow_id = :id AND revision = :expected
            """)
        .param("current", updated.currentState().name())
        .param("previous", nameOrNull(updated.previousState()))
        .param("resume", nameOrNull(updated.resumeState()))
        .param("revision", updated.revision())
        .param("status", updated.status().name())
        .param("leaseOwner", updated.leaseOwner())
        .param("leaseUntil", WorkItemRepository.ts(updated.leaseUntil()))
        .param("updated", WorkItemRepository.ts(updated.updatedAt()))
        .param("id", updated.workflowId())
        .param("expected", expectedRevision)
        .update();
  }

  /**
   * 만료됐거나 없는 리스를 이 노드가 가져간다.
   *
   * @param workflowId 대상
   * @param owner 이 서버 식별자. 비면 안 된다
   * @param until 만료 시각 (UTC)
   * @param now 비교 기준 시각 (UTC)
   * @return 획득하면 true
   */
  public boolean tryAcquireLease(String workflowId, String owner, Instant until, Instant now) {
    int rows =
        jdbc.sql(
                """
                UPDATE oncode.workflow
                SET lease_owner = :owner,
                    lease_until = :until,
                    updated_at = :now
                WHERE workflow_id = :id
                  AND (lease_owner IS NULL
                       OR lease_until IS NULL
                       OR lease_until < :now)
                """)
            .param("owner", owner)
            .param("until", Timestamp.from(until))
            .param("now", Timestamp.from(now))
            .param("id", workflowId)
            .update();
    return rows == 1;
  }

  /**
   * 리스를 비운다. WAITING 진입 또는 핸들러 종료 시 호출한다.
   *
   * @param workflowId 대상
   * @param now 갱신 시각 (UTC)
   */
  public void clearLease(String workflowId, Instant now) {
    jdbc.sql(
            """
            UPDATE oncode.workflow
            SET lease_owner = NULL, lease_until = NULL, updated_at = :now
            WHERE workflow_id = :id
            """)
        .param("now", Timestamp.from(now))
        .param("id", workflowId)
        .update();
  }

  private static WorkflowRecord mapRow(java.sql.ResultSet rs) throws java.sql.SQLException {
    return new WorkflowRecord(
        rs.getString("workflow_id"),
        rs.getString("work_item_id"),
        rs.getString("workflow_type"),
        WorkflowState.valueOf(rs.getString("current_state")),
        parseState(rs.getString("previous_state")),
        parseState(rs.getString("resume_state")),
        rs.getInt("revision"),
        WorkflowStatus.valueOf(rs.getString("status")),
        rs.getInt("retry_count"),
        rs.getString("lease_owner"),
        tsOrNull(rs.getTimestamp("lease_until")),
        rs.getTimestamp("created_at").toInstant(),
        rs.getTimestamp("updated_at").toInstant());
  }

  private static WorkflowState parseState(String value) {
    return value == null ? null : WorkflowState.valueOf(value);
  }

  private static Instant tsOrNull(Timestamp value) {
    return value == null ? null : value.toInstant();
  }

  private static String nameOrNull(WorkflowState state) {
    return state == null ? null : state.name();
  }
}
