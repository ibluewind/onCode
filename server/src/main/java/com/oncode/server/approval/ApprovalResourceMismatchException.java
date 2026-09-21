package com.oncode.server.approval;

/**
 * 승인 종류 또는 리소스가 현재 대기 상태와 맞지 않을 때 던진다.
 * 호출자는 워크플로 상태를 바꾸지 않은 것으로 본다.
 */
public class ApprovalResourceMismatchException extends RuntimeException {

  /**
   * 불일치 이유를 담는다.
   *
   * @param message 사람 읽기용 이유. 비면 안 된다
   */
  public ApprovalResourceMismatchException(String message) {
    super(message);
  }
}
