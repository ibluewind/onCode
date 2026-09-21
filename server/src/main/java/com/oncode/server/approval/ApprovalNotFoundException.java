package com.oncode.server.approval;

/**
 * 지정한 승인 행이 없을 때 던진다.
 */
public class ApprovalNotFoundException extends RuntimeException {

  /**
   * 없는 승인 식별자를 담는다.
   *
   * @param approvalId APR-...
   */
  public ApprovalNotFoundException(String approvalId) {
    super("approval not found: " + approvalId);
  }
}
