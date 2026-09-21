package com.oncode.server.persistence;

import java.time.Instant;

/**
 * {@code context_entry} 행. 에이전트 간 직접 페이로드 전달 대신 이 행을 참조한다.
 *
 * @param contextRef {@code ctx://...} URI
 * @param workItemId 소속 워크 아이템
 * @param refType 종류 (DESIGN 등)
 * @param payloadJson JSON 본문. hidden reasoning 금지
 * @param createdAt 최초 저장 (UTC)
 * @param updatedAt 마지막 put (UTC)
 */
public record ContextEntryRecord(
    String contextRef,
    String workItemId,
    String refType,
    String payloadJson,
    Instant createdAt,
    Instant updatedAt) {}
