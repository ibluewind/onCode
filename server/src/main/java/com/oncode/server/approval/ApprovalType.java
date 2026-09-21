package com.oncode.server.approval;

/**
 * 승인 종류. 설계 승인과 코드 변경 승인은 서로 바꿔 쓸 수 없다.
 */
public enum ApprovalType {
  DESIGN,
  CODE_CHANGE
}
