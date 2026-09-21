package com.oncode.server.workflow;

/**
 * 같은 워크플로를 두 노드가 동시에 전이하려 revision이 어긋날 때 던진다.
 */
public class WorkflowConcurrentModificationException extends RuntimeException {

  /**
   * 충돌한 워크플로를 표시한다.
   *
   * @param workflowId WF-...
   * @param expectedRevision 읽었던 버전
   */
  public WorkflowConcurrentModificationException(String workflowId, int expectedRevision) {
    super("workflow revision conflict: " + workflowId + " expected=" + expectedRevision);
  }
}
