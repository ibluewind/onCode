package com.oncode.server.persistence;

import com.oncode.server.workflow.WorkItemStatus;
import java.sql.Timestamp;
import java.time.Instant;
import org.springframework.jdbc.core.simple.JdbcClient;
import org.springframework.stereotype.Repository;

/**
 * {@code work_item} 삽입·조회. 워크플로 전이는 하지 않는다.
 */
@Repository
public class WorkItemRepository {

  private final JdbcClient jdbc;

  /**
   * JDBC 클라이언트를 받는다.
   *
   * @param jdbc 기본 DataSource의 {@link JdbcClient}. null이면 안 된다
   */
  public WorkItemRepository(JdbcClient jdbc) {
    this.jdbc = jdbc;
  }

  /**
   * 새 워크 아이템 행을 삽입한다.
   *
   * @param record 저장할 행. {@code workItemId}와 {@code originalRequest}는 비어 있으면 안 된다
   */
  public void insert(WorkItemRecord record) {
    jdbc.sql(
            """
            INSERT INTO oncode.work_item (
              work_item_id, title, original_request, work_item_type, status, created_at, updated_at
            ) VALUES (
              :id, :title, :request, :type, :status, :created, :updated
            )
            """)
        .param("id", record.workItemId())
        .param("title", record.title())
        .param("request", record.originalRequest())
        .param("type", record.workItemType())
        .param("status", record.status().name())
        .param("created", Timestamp.from(record.createdAt()))
        .param("updated", Timestamp.from(record.updatedAt()))
        .update();
  }

  /**
   * 식별자로 한 행을 읽는다.
   *
   * @param workItemId WI-... . 빈 값이면 결과가 없다
   * @return 없으면 empty
   */
  public java.util.Optional<WorkItemRecord> findById(String workItemId) {
    return jdbc.sql(
            """
            SELECT work_item_id, title, original_request, work_item_type, status, created_at, updated_at
            FROM oncode.work_item
            WHERE work_item_id = :id
            """)
        .param("id", workItemId)
        .query(
            (rs, rowNum) ->
                new WorkItemRecord(
                    rs.getString("work_item_id"),
                    rs.getString("title"),
                    rs.getString("original_request"),
                    rs.getString("work_item_type"),
                    WorkItemStatus.valueOf(rs.getString("status")),
                    rs.getTimestamp("created_at").toInstant(),
                    rs.getTimestamp("updated_at").toInstant()))
        .optional();
  }

  /**
   * 상태를 갱신한다. 워크플로 전이는 하지 않는다.
   *
   * @param workItemId WI-...
   * @param status 새 상태. null 불가
   * @param updatedAt 갱신 시각. null 불가
   * @return 갱신된 행 수
   */
  public int updateStatus(String workItemId, WorkItemStatus status, Instant updatedAt) {
    return jdbc.sql(
            """
            UPDATE oncode.work_item
            SET status = :status, updated_at = :updated
            WHERE work_item_id = :id
            """)
        .param("id", workItemId)
        .param("status", status.name())
        .param("updated", Timestamp.from(updatedAt))
        .update();
  }

  /**
   * Instant를 JDBC Timestamp로 바꾼다. null이면 null.
   *
   * @param value UTC 시각
   * @return JDBC 값
   */
  static Timestamp ts(Instant value) {
    return value == null ? null : Timestamp.from(value);
  }
}
