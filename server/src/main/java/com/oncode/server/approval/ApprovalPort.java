package com.oncode.server.approval;

import java.time.Instant;
import java.util.Optional;

/**
 * 설계/코드 승인 포트. 테스트 어댑터가 이 경계를 구현한다.
 * 워크스페이스 apply는 호출하지 않는다.
 */
public interface ApprovalPort {

  /**
   * 대기 중 승인 행을 만든다. 같은 워크플로·종류의 REQUESTED가 있으면 insert가 실패할 수 있다.
   *
   * @param workItemId WI-... . 공백 불가
   * @param workflowId WF-... . 공백 불가
   * @param type DESIGN 또는 CODE_CHANGE
   * @param resourceType DESIGN_REF 또는 CHANGE_SET
   * @param resourceId 바인딩할 리소스. 공백 불가
   * @param expiresAt 만료 시각 (UTC). null이면 만료 없음
   * @return 저장된 REQUESTED 행
   */
  ApprovalRecord request(
      String workItemId,
      String workflowId,
      ApprovalType type,
      String resourceType,
      String resourceId,
      Instant expiresAt);

  /**
   * 식별자로 읽는다.
   *
   * @param approvalId APR-...
   * @return 없으면 empty
   */
  Optional<ApprovalRecord> findById(String approvalId);

  /**
   * 워크플로의 REQUESTED 승인을 종류로 찾는다.
   *
   * @param workflowId WF-...
   * @param type DESIGN 또는 CODE_CHANGE
   * @return 없으면 empty
   */
  Optional<ApprovalRecord> findRequested(String workflowId, ApprovalType type);

  /**
   * 아직 쓸 수 있는 REQUESTED 행인지 본다. 만료면 EXPIRED로 표시한다.
   *
   * @param approvalId APR-...
   * @return 사용 가능한 행
   * @throws ApprovalNotFoundException 없을 때
   * @throws ApprovalExpiredException 만료됐을 때
   * @throws ApprovalReuseException 이미 쓰였거나 거절됐을 때
   */
  ApprovalRecord requireUsable(String approvalId);

  /**
   * 승인을 1회 소비한다. 워크플로 전이는 호출자가 한다.
   *
   * @param approvalId APR-...
   * @param decision APPROVE 또는 REJECT
   * @param actor 결정 주체. null 가능
   * @param reason 사유. null 가능
   * @return 소비된 행
   */
  ApprovalRecord consume(String approvalId, ApprovalDecision decision, String actor, String reason);
}
