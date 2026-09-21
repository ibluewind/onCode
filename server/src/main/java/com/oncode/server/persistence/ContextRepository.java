package com.oncode.server.persistence;

import java.util.Optional;
import org.springframework.jdbc.core.simple.JdbcClient;
import org.springframework.stereotype.Repository;

/**
 * {@code context_entry} 저장·조회. SQL을 에이전트에 노출하지 않는다.
 */
@Repository
public class ContextRepository {

  private final JdbcClient jdbc;

  /**
   * JDBC 클라이언트를 받는다.
   *
   * @param jdbc 기본 DataSource. null이면 안 된다
   */
  public ContextRepository(JdbcClient jdbc) {
    this.jdbc = jdbc;
  }

  /**
   * 같은 {@code context_ref}면 payload를 덮어쓴다.
   *
   * @param record 저장할 행. ref와 payload는 비어 있으면 안 된다
   */
  public void upsert(ContextEntryRecord record) {
    jdbc.sql(
            """
            INSERT INTO oncode.context_entry (
              context_ref, work_item_id, ref_type, payload, created_at, updated_at
            ) VALUES (
              :ref, :workItemId, :refType, CAST(:payload AS jsonb), :created, :updated
            )
            ON CONFLICT (context_ref) DO UPDATE SET
              payload = CAST(:payload AS jsonb),
              updated_at = :updated
            """)
        .param("ref", record.contextRef())
        .param("workItemId", record.workItemId())
        .param("refType", record.refType())
        .param("payload", record.payloadJson())
        .param("created", WorkItemRepository.ts(record.createdAt()))
        .param("updated", WorkItemRepository.ts(record.updatedAt()))
        .update();
  }

  /**
   * ref로 한 건을 읽는다.
   *
   * @param contextRef {@code ctx://...}
   * @return 없으면 empty
   */
  public Optional<ContextEntryRecord> findByRef(String contextRef) {
    return jdbc.sql(
            """
            SELECT context_ref, work_item_id, ref_type, payload::text AS payload, created_at, updated_at
            FROM oncode.context_entry
            WHERE context_ref = :ref
            """)
        .param("ref", contextRef)
        .query(
            (rs, rowNum) ->
                new ContextEntryRecord(
                    rs.getString("context_ref"),
                    rs.getString("work_item_id"),
                    rs.getString("ref_type"),
                    rs.getString("payload"),
                    rs.getTimestamp("created_at").toInstant(),
                    rs.getTimestamp("updated_at").toInstant()))
        .optional();
  }
}
