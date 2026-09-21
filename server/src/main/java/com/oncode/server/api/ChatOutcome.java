package com.oncode.server.api;

import java.util.List;

/**
 * 채팅 접수 결과. proto 타입이 아니다.
 */
public sealed interface ChatOutcome {

  /**
   * work item을 만들거나 재개했고 진행 이벤트를 보낼 수 있다.
   *
   * @param workItemId WI-...
   * @param workflowId WF-...
   * @param progress 순서대로 보낼 진행
   * @param pending 대기 승인. 없으면 null. apply 권한 아님
   * @param results 빌드/테스트 요약. 없으면 빈 목록
   */
  record Accepted(
      String workItemId,
      String workflowId,
      List<ProgressEvent> progress,
      ApprovalView pending,
      List<RunResult> results)
      implements ChatOutcome {

    /**
     * 승인·결과가 없는 접수. 슬라이스 2 호출부 호환.
     *
     * @param workItemId WI-...
     * @param workflowId WF-...
     * @param progress 진행 이벤트
     */
    public Accepted(String workItemId, String workflowId, List<ProgressEvent> progress) {
      this(workItemId, workflowId, progress, null, List.of());
    }
  }

  /**
   * 파일 경로 등이 부족해 사용자 질문이 필요하다.
   *
   * @param question IDE에 띄울 질문
   */
  record NeedQuestion(Question question) implements ChatOutcome {}

  /**
   * 소스 첨부 등 fail-closed 거부.
   *
   * @param code 안정적 코드
   * @param message 설명
   */
  record Rejected(String code, String message) implements ChatOutcome {}
}
