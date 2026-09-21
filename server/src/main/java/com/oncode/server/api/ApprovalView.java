package com.oncode.server.api;

/**
 * IDE에 보낼 대기 승인. proto가 아니며 apply 권한이 아니다.
 *
 * @param kind DESIGN 또는 CODE_CHANGE
 * @param approvalId APR-...
 * @param workItemId WI-...
 * @param workflowId WF-...
 * @param workspaceId 바인딩된 워크스페이스. 비면 안 된다
 * @param resourceType DESIGN_REF 또는 CHANGE_SET
 * @param resourceId 승인한 리소스
 * @param designIdentity 설계 본문 해시. DESIGN일 때만
 * @param designSummary 사용자용 설계 요약. DESIGN일 때만
 * @param proposedChangesJson ProposedChanges JSON. CODE_CHANGE일 때만. unified diff 아님
 * @param canApply 항상 false. 승인만으로 apply 하지 않는다
 */
public record ApprovalView(
    String kind,
    String approvalId,
    String workItemId,
    String workflowId,
    String workspaceId,
    String resourceType,
    String resourceId,
    String designIdentity,
    String designSummary,
    String proposedChangesJson,
    boolean canApply) {

  public static final String DESIGN = "DESIGN";
  public static final String CODE_CHANGE = "CODE_CHANGE";

  /**
   * 설계 대기 화면용 뷰. diff와 apply는 없다.
   */
  public static ApprovalView design(
      String approvalId,
      String workItemId,
      String workflowId,
      String workspaceId,
      String resourceType,
      String resourceId,
      String designIdentity,
      String designSummary) {
    return new ApprovalView(
        DESIGN,
        approvalId,
        workItemId,
        workflowId,
        workspaceId,
        resourceType,
        resourceId,
        designIdentity,
        designSummary,
        null,
        false);
  }

  /**
   * 코드 승인 준비. Local Agent가 actual diff를 만들기 전의 ProposedChanges다.
   */
  public static ApprovalView codeProposed(
      String approvalId,
      String workItemId,
      String workflowId,
      String workspaceId,
      String resourceType,
      String resourceId,
      String proposedChangesJson) {
    return new ApprovalView(
        CODE_CHANGE,
        approvalId,
        workItemId,
        workflowId,
        workspaceId,
        resourceType,
        resourceId,
        null,
        null,
        proposedChangesJson,
        false);
  }
}
