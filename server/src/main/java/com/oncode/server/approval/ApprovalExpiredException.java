package com.oncode.server.approval;

/**
 * 만료된 승인을 쓰려 할 때 던진다. 호출자는 상태를 바꾸지 않은 것으로 본다.
 */
public class ApprovalExpiredException extends RuntimeException {

  /**
   * 만료된 승인을 표시한다.
   *
   * @param approvalId APR-...
   */
  public ApprovalExpiredException(String approvalId) {
    super("approval expired: " + approvalId);
  }
}
