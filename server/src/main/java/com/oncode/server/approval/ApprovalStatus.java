package com.oncode.server.approval;

/**
 * 승인 행의 수명. REQUESTED만 결정에 사용할 수 있다.
 */
public enum ApprovalStatus {
  REQUESTED,
  APPROVED,
  REJECTED,
  EXPIRED,
  CONSUMED
}
