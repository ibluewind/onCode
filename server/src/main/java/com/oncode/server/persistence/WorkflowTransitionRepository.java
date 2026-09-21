package com.oncode.server.persistence;

import com.oncode.server.workflow.WorkflowState;
import java.util.List;
import org.springframework.jdbc.core.simple.JdbcClient;
import org.springframework.stereotype.Repository;

/**
 * {@code workflow_transition} 삽입·조회. 전이 허용 여부는 검사하지 않는다.
 */
@Repository
public class WorkflowTransitionRepository {

  private final JdbcClient jdbc;

  /**
   * JDBC 클라이언트를 받는다.
   *
   * @param jdbc 기본 DataSource. null이면 안 된다
   */
  public WorkflowTransitionRepository(JdbcClient jdbc) {
    this.jdbc = jdbc;
  }

  /**
   * 전이 이력 한 행을 추가한다.
   *
   * @param record 이미 발급된 {@code transitionId}를 가진 행
   */
  public void insert(WorkflowTransitionRecord record) {
    jdbc.sql(
            """
            INSERT INTO oncode.workflow_transition (
              transition_id, workflow_id, from_state, to_state, trigger_type, trigger_id, reason, created_at
            ) VALUES (
              :id, :workflowId, :fromState, :toState, :triggerType, :triggerId, :reason, :created
            )
            """)
        .param("id", record.transitionId())
        .param("workflowId", record.workflowId())
        .param("fromState", record.fromState() == null ? null : record.fromState().name())
        .param("toState", record.toState().name())
        .param("triggerType", record.triggerType())
        .param("triggerId", record.triggerId())
        .param("reason", record.reason())
        .param("created", WorkItemRepository.ts(record.createdAt()))
        .update();
  }

  /**
   * 워크플로의 전이를 시간순으로 읽는다.
   *
   * @param workflowId WF-...
   * @return 없으면 빈 리스트. null 아님
   */
  public List<WorkflowTransitionRecord> findByWorkflowId(String workflowId) {
    return jdbc.sql(
            """
            SELECT transition_id, workflow_id, from_state, to_state, trigger_type, trigger_id, reason, created_at
            FROM oncode.workflow_transition
            WHERE workflow_id = :id
            ORDER BY created_at ASC, transition_id ASC
            """)
        .param("id", workflowId)
        .query(
            (rs, rowNum) ->
                new WorkflowTransitionRecord(
                    rs.getString("transition_id"),
                    rs.getString("workflow_id"),
                    parseState(rs.getString("from_state")),
                    WorkflowState.valueOf(rs.getString("to_state")),
                    rs.getString("trigger_type"),
                    rs.getString("trigger_id"),
                    rs.getString("reason"),
                    rs.getTimestamp("created_at").toInstant()))
        .list();
  }

  private static WorkflowState parseState(String value) {
    return value == null ? null : WorkflowState.valueOf(value);
  }
}
