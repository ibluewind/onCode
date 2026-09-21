package com.oncode.server.workflow;

/**
 * 허용되지 않은 공식 상태 전이를 거부할 때 던진다.
 * 호출자는 상태를 바꾸지 않은 것으로 본다.
 */
public class IllegalWorkflowTransitionException extends RuntimeException {

  /**
   * 거부된 전이를 설명한다.
   *
   * @param message 사람 읽기용 이유. 빈 문자열이면 안 된다
   */
  public IllegalWorkflowTransitionException(String message) {
    super(message);
  }
}
