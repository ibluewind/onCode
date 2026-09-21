package com.oncode.server.workflow;

/**
 * 지정한 워크플로 행이 PostgreSQL에 없을 때 던진다.
 */
public class WorkflowNotFoundException extends RuntimeException {

  /**
   * 없는 워크플로 식별자를 담는다.
   *
   * @param workflowId 조회에 사용한 ID. 빈 값이면 안 된다
   */
  public WorkflowNotFoundException(String workflowId) {
    super("workflow not found: " + workflowId);
  }
}
