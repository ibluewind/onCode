package com.oncode.server.context;

/**
 * {@code context.get} 대상 ref가 없을 때 던진다.
 */
public class ContextNotFoundException extends RuntimeException {

  /**
   * 없는 참조를 담는다.
   *
   * @param contextRef {@code ctx://...}. 빈 값이면 안 된다
   */
  public ContextNotFoundException(String contextRef) {
    super("context not found: " + contextRef);
  }
}
