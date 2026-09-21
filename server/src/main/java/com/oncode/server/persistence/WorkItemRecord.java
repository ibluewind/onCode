package com.oncode.server.persistence;

import com.oncode.server.workflow.WorkItemStatus;
import java.time.Instant;

/**
 * {@code work_item} 행. 워크플로 공식 상태를 담지 않는다.
 *
 * @param workItemId 식별자 (WI-...)
 * @param title 요약. 없으면 null
 * @param originalRequest 사용자 원문. 비어 있으면 안 된다
 * @param workItemType 요청 유형
 * @param status 수명 상태
 * @param createdAt 생성 시각 (UTC)
 * @param updatedAt 변경 시각 (UTC)
 */
public record WorkItemRecord(
    String workItemId,
    String title,
    String originalRequest,
    String workItemType,
    WorkItemStatus status,
    Instant createdAt,
    Instant updatedAt) {}
