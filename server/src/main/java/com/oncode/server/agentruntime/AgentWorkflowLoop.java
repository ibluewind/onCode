package com.oncode.server.agentruntime;

import com.oncode.server.approval.ApprovalDecision;
import com.oncode.server.approval.ApprovalPort;
import com.oncode.server.approval.ApprovalRecord;
import com.oncode.server.approval.ApprovalType;
import com.oncode.server.persistence.WorkflowRecord;
import com.oncode.server.workflow.IllegalWorkflowTransitionException;
import com.oncode.server.workflow.WorkflowEngine;
import com.oncode.server.workflow.WorkflowState;
import com.oncode.server.workflow.WorkflowTransitions;
import java.util.List;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

/**
 * mock 프로젝트 컨텍스트로 Design → Approval → Implementation 최소 루프를 돌린다.
 * APPLYING_CHANGE에서 Local Agent apply는 호출하지 않는다.
 */
@Service
public class AgentWorkflowLoop {

  private final WorkflowEngine engine;
  private final AgentRuntime agents;
  private final ApprovalPort approvals;

  /**
   * 엔진과 런타임을 받는다.
   *
   * @param engine 워크플로 SSOT. null 불가
   * @param agents 세 에이전트 런타임. null 불가
   * @param approvals 대기 승인 조회. null 불가
   */
  public AgentWorkflowLoop(WorkflowEngine engine, AgentRuntime agents, ApprovalPort approvals) {
    this.engine = engine;
    this.agents = agents;
    this.approvals = approvals;
  }

  /**
   * 요청을 받아 Senior가 설계를 남기고 설계 승인 대기까지 간다.
   *
   * @param originalRequest 사용자 원문
   * @return WAITING_DESIGN_APPROVAL 행
   */
  @Transactional
  public WorkflowRecord startDesignApproval(String originalRequest) {
    WorkflowRecord current = engine.receive(originalRequest);
    current = driveLinearTo(current, WorkflowState.DESIGNING);
    agents.run(
        AgentId.SENIOR_DEVELOPER,
        new AgentTurn(
            current.workItemId(), current.workflowId(), List.of(), originalRequest));
    return engine.transition(
        current.workflowId(), WorkflowState.WAITING_DESIGN_APPROVAL, "senior-design");
  }

  /**
   * IMPLEMENTING에서 Implementation propose와 Review를 돌리고 코드 승인 대기로 간다.
   *
   * @param workflowId 현재 IMPLEMENTING인 WF-...
   * @return WAITING_CODE_APPROVAL 행
   */
  @Transactional
  public WorkflowRecord implementAndReview(String workflowId) {
    WorkflowRecord current = engine.require(workflowId);
    if (current.currentState() != WorkflowState.IMPLEMENTING) {
      throw new IllegalWorkflowTransitionException(
          "implementAndReview requires IMPLEMENTING, was " + current.currentState());
    }
    String designRef = WorkflowEngine.designResourceId(current.workItemId());
    agents.run(
        AgentId.IMPLEMENTATION,
        new AgentTurn(
            current.workItemId(),
            current.workflowId(),
            List.of(designRef),
            "implement the approved design"));
    current =
        engine.transition(current.workflowId(), WorkflowState.REVIEWING, "implementation-done");
    String changeSetRef =
        "ctx://work-items/" + current.workItemId() + "/change_set/latest";
    agents.run(
        AgentId.REVIEW,
        new AgentTurn(
            current.workItemId(),
            current.workflowId(),
            List.of(changeSetRef),
            "review the proposed change set"));
    current =
        engine.transition(
            current.workflowId(), WorkflowState.PREPARING_CHANGE, "review-done");
    return engine.transition(
        current.workflowId(), WorkflowState.WAITING_CODE_APPROVAL, "await-code-approval");
  }

  /**
   * 설계 승인을 적용한다. 테스트 어댑터용.
   *
   * @param workflowId WAITING_DESIGN_APPROVAL
   * @return IMPLEMENTING 행
   */
  @Transactional
  public WorkflowRecord approveDesign(String workflowId) {
    ApprovalRecord pending =
        approvals
            .findRequested(workflowId, ApprovalType.DESIGN)
            .orElseThrow(() -> new IllegalStateException("no DESIGN approval"));
    return engine.decide(pending.approvalId(), ApprovalDecision.APPROVE, "user", "approve-design");
  }

  private WorkflowRecord driveLinearTo(WorkflowRecord current, WorkflowState stopAt) {
    WorkflowRecord cursor = current;
    while (cursor.currentState() != stopAt) {
      WorkflowState from = cursor.currentState();
      WorkflowState next =
          WorkflowTransitions.nextLinear(from)
              .orElseThrow(
                  () -> new IllegalWorkflowTransitionException("no linear next from " + from));
      cursor = engine.transition(cursor.workflowId(), next, "linear-advance");
    }
    return cursor;
  }
}
