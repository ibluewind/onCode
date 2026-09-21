package com.oncode.server.approval;

/**
 * 이미 쓰였거나 거절·만료된 승인을 다시 쓰려 할 때 던진다.
 * 호출자는 워크플로 상태를 바꾸지 않은 것으로 본다.
 */
public class ApprovalReuseException extends RuntimeException {

  /**
   * 재사용이 거부된 승인을 표시한다.
   *
   * @param approvalId APR-...
   * @param status 현재 상태
   */
  public ApprovalReuseException(String approvalId, ApprovalStatus status) {
    super("approval cannot be reused: " + approvalId + " status=" + status);
  }
}
