package com.oncode.server.agentruntime;

import static org.assertj.core.api.Assertions.assertThat;

import com.oncode.server.approval.ApprovalDecision;
import com.oncode.server.approval.ApprovalPort;
import com.oncode.server.approval.ApprovalType;
import com.oncode.server.context.ContextService;
import com.oncode.server.persistence.WorkflowRecord;
import com.oncode.server.support.PostgresSpringBootTest;
import com.oncode.server.toolgateway.InMemoryLocalAgentChannel;
import com.oncode.server.workflow.WorkflowEngine;
import com.oncode.server.workflow.WorkflowState;
import com.oncode.server.workflow.WorkflowStatus;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;

/**
 * User Request → Design → Approval → Implementation mock 루프를 검증한다.
 */
@PostgresSpringBootTest
class AgentWorkflowLoopTest {

  @Autowired private AgentWorkflowLoop loop;
  @Autowired private WorkflowEngine engine;
  @Autowired private ApprovalPort approvals;
  @Autowired private ContextService contexts;
  @Autowired private InMemoryLocalAgentChannel harness;

  /**
   * Senior 설계 승인 후 Implementation이 propose하고 Review가 따라간다. apply는 없다.
   */
  @Test
  void designApproveImplementationLoop() {
    WorkflowRecord waiting = loop.startDesignApproval("로그인 실패 5회 시 계정을 잠가줘.");
    assertThat(waiting.currentState()).isEqualTo(WorkflowState.WAITING_DESIGN_APPROVAL);
    String design = contexts.get(WorkflowEngine.designResourceId(waiting.workItemId()));
    assertThat(design).contains("summary");

    WorkflowRecord implementing = loop.approveDesign(waiting.workflowId());
    assertThat(implementing.currentState()).isEqualTo(WorkflowState.IMPLEMENTING);

    WorkflowRecord codeWaiting = loop.implementAndReview(implementing.workflowId());
    assertThat(codeWaiting.currentState()).isEqualTo(WorkflowState.WAITING_CODE_APPROVAL);
    assertThat(codeWaiting.status()).isEqualTo(WorkflowStatus.WAITING);
    assertThat(contexts.get("ctx://work-items/" + waiting.workItemId() + "/change_set/latest"))
        .contains("change_set_id");
    assertThat(contexts.get("ctx://work-items/" + waiting.workItemId() + "/review/latest"))
        .contains("verdict");
    assertThat(harness.invokedTools()).doesNotContain("workspace.apply_changes");
    assertThat(approvals.findRequested(codeWaiting.workflowId(), ApprovalType.CODE_CHANGE))
        .isPresent();

    WorkflowRecord stillWaiting = engine.resumeAfterRestart(codeWaiting.workflowId());
    assertThat(stillWaiting.currentState()).isEqualTo(WorkflowState.WAITING_CODE_APPROVAL);

    var codeApproval =
        approvals.findRequested(codeWaiting.workflowId(), ApprovalType.CODE_CHANGE).orElseThrow();
    WorkflowRecord applying =
        engine.decide(codeApproval.approvalId(), ApprovalDecision.APPROVE, "user", "ok");
    assertThat(applying.currentState()).isEqualTo(WorkflowState.APPLYING_CHANGE);
    assertThat(harness.invokedTools()).doesNotContain("workspace.apply_changes");
  }
}
