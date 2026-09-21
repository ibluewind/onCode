package com.oncode.server.context;

import com.oncode.server.persistence.ContextEntryRecord;
import com.oncode.server.persistence.ContextRepository;
import com.oncode.server.persistence.WorkItemRepository;
import java.time.Clock;
import java.time.Instant;
import java.util.Locale;
import java.util.Objects;
import java.util.Optional;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

/**
 * Context Storage의 get/put. 에이전트는 SQL이 아니라 {@code context_ref}만 교환한다.
 * {@code context.query}와 대용량 검색은 구현하지 않는다.
 */
@Service
public class ContextService {

  private final ContextRepository contexts;
  private final WorkItemRepository workItems;
  private final Clock clock;

  /**
   * 저장소와 시계를 받는다.
   *
   * @param contexts 컨텍스트 행 저장소. null 불가
   * @param workItems 워크 아이템 존재 확인용. null 불가
   */
  @Autowired
  public ContextService(ContextRepository contexts, WorkItemRepository workItems) {
    this(contexts, workItems, Clock.systemUTC());
  }

  /**
   * 테스트용 시계 주입.
   *
   * @param contexts 컨텍스트 저장소
   * @param workItems 워크 아이템 저장소
   * @param clock UTC 시계
   */
  ContextService(ContextRepository contexts, WorkItemRepository workItems, Clock clock) {
    this.contexts = contexts;
    this.workItems = workItems;
    this.clock = clock;
  }

  /**
   * JSON 본문을 저장하고 참조 URI를 돌려준다. 같은 타입의 {@code /latest}는 덮어쓴다.
   *
   * @param workItemId 존재하는 WI-... . 없으면 예외
   * @param refType DESIGN 등. 공백이면 안 된다
   * @param payloadJson 구조화 JSON 문자열. null이면 안 된다
   * @return {@code ctx://work-items/{id}/{type}/latest}
   */
  @Transactional
  public String put(String workItemId, String refType, String payloadJson) {
    if (workItemId == null || workItemId.isBlank()) {
      throw new IllegalArgumentException("workItemId must not be blank");
    }
    if (refType == null || refType.isBlank()) {
      throw new IllegalArgumentException("refType must not be blank");
    }
    if (payloadJson == null || payloadJson.isBlank()) {
      throw new IllegalArgumentException("payloadJson must not be blank");
    }
    workItems
        .findById(workItemId)
        .orElseThrow(() -> new IllegalArgumentException("unknown work item: " + workItemId));
    String typeKey = refType.trim().toLowerCase(Locale.ROOT);
    String ref = "ctx://work-items/" + workItemId + "/" + typeKey + "/latest";
    Instant now = clock.instant();
    contexts.upsert(
        new ContextEntryRecord(ref, workItemId, refType.trim(), payloadJson, now, now));
    return ref;
  }

  /**
   * 참조 URI로 본문을 읽는다. 없으면 empty. 트랜잭션을 롤백하지 않는다.
   *
   * @param contextRef {@code ctx://...}
   * @return 저장된 JSON. 없거나 공백 ref면 empty
   */
  public Optional<String> find(String contextRef) {
    if (contextRef == null || contextRef.isBlank()) {
      return Optional.empty();
    }
    return contexts.findByRef(contextRef).map(ContextEntryRecord::payloadJson);
  }

  /**
   * 참조 URI로 본문을 읽는다.
   *
   * @param contextRef {@code ctx://...}. 공백이면 안 된다
   * @return 저장된 JSON 문자열
   * @throws ContextNotFoundException 행이 없을 때
   */
  public String get(String contextRef) {
    Objects.requireNonNull(contextRef, "contextRef");
    if (contextRef.isBlank()) {
      throw new IllegalArgumentException("contextRef must not be blank");
    }
    return find(contextRef).orElseThrow(() -> new ContextNotFoundException(contextRef));
  }

  /**
   * 참조가 저장돼 있으면 true. 없으면 예외를 던지지 않는다.
   *
   * @param contextRef {@code ctx://...}
   * @return 행이 있으면 true
   */
  public boolean exists(String contextRef) {
    if (contextRef == null || contextRef.isBlank()) {
      return false;
    }
    return contexts.findByRef(contextRef).isPresent();
  }
}
