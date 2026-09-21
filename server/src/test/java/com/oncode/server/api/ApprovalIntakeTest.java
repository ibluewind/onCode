package com.oncode.server.api;

import static org.assertj.core.api.Assertions.assertThat;

import com.oncode.server.support.PostgresSpringBootTest;
import com.oncode.server.toolgateway.InMemoryLocalAgentChannel;
import com.oncode.server.workflow.WorkflowEngine;
import com.oncode.server.workflow.WorkflowState;
import java.util.HashMap;
import java.util.Map;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;

/**
 * 설계/코드 승인이 바인딩되고, 거절은 apply 없이 멈추며, 재연결이 대기를 복원하는지 검증한다.
 */
@PostgresSpringBootTest
class ApprovalIntakeTest {

  @Autowired private ChatIntake chat;
  @Autowired private ApprovalIntake intake;
  @Autowired private WorkflowEngine engine;
  @Autowired private InMemoryLocalAgentChannel harness;

  /**
   * 경로가 있는 채팅은 설계 대기와 설계 리뷰 뷰를 만든다. apply 플래그는 없다.
   */
  @Test
  void chatYieldsDesignReviewWithoutApply() {
    ChatOutcome.Accepted accepted = acceptChat();
    assertThat(accepted.pending()).isNotNull();
    assertThat(accepted.pending().kind()).isEqualTo(ApprovalView.DESIGN);
    assertThat(accepted.pending().canApply()).isFalse();
    assertThat(accepted.pending().approvalId()).startsWith("APR-");
    assertThat(accepted.pending().designIdentity()).startsWith("sha256:");
    assertThat(accepted.pending().proposedChangesJson()).isNull();
  }

  /**
   * 설계 거절은 REJECTED이고 코드 승인·apply가 없다.
   */
  @Test
  void designRejectStopsWithoutApply() {
    ChatOutcome.Accepted created = acceptChat();
    ChatOutcome outcome = intake.decide(decision(created.pending(), "REJECT", null));
    assertThat(outcome).isInstanceOf(ChatOutcome.Accepted.class);
    ChatOutcome.Accepted rejected = (ChatOutcome.Accepted) outcome;
    assertThat(rejected.pending()).isNull();
    assertThat(engine.require(rejected.workflowId()).currentState())
        .isEqualTo(WorkflowState.REJECTED);
    assertThat(harness.invokedTools()).doesNotContain("workspace.apply_changes");
    assertThat(rejected.results()).isEmpty();
  }

  /**
   * 설계 승인은 구현 루프를 돌리고 코드 ProposedChanges를 남긴다. actual diff와 apply는 없다.
   */
  @Test
  void designApproveOpensCodeProposedWithoutApply() {
    ChatOutcome.Accepted created = acceptChat();
    ChatOutcome outcome = intake.decide(decision(created.pending(), "APPROVE", null));
    ChatOutcome.Accepted next = (ChatOutcome.Accepted) outcome;
    assertThat(engine.require(next.workflowId()).currentState())
        .isEqualTo(WorkflowState.WAITING_CODE_APPROVAL);
    assertThat(next.pending().kind()).isEqualTo(ApprovalView.CODE_CHANGE);
    assertThat(next.pending().canApply()).isFalse();
    assertThat(next.pending().proposedChangesJson()).contains("change_set_id");
    assertThat(next.pending().proposedChangesJson()).doesNotContain("--- a/");
    assertThat(harness.invokedTools()).contains("workspace.propose_changes");
    assertThat(harness.invokedTools()).doesNotContain("workspace.apply_changes");
  }

  /**
   * 소비된 설계 승인 ID로 bind해도 트랜잭션이 rollback-only가 되지 않는다.
   */
  @Test
  void bindDiffConsumedDesignApprovalIsRejected() {
    ChatOutcome.Accepted created = acceptChat();
    String designApr = created.pending().approvalId();
    ChatOutcome.Accepted code = afterDesignApproveFrom(created);
    Map<String, Object> bind = decision(code.pending(), "APPROVE", "sha256:agent-diff");
    bind.put("approval_id", designApr);
    ChatOutcome outcome = intake.bindDiff(bind);
    assertThat(outcome).isInstanceOf(ChatOutcome.Rejected.class);
    assertThat(engine.require(code.workflowId()).currentState())
        .isEqualTo(WorkflowState.WAITING_CODE_APPROVAL);
  }

  /**
   * 없는 승인 ID bind는 Rejected이고 워크플로는 그대로다.
   */
  @Test
  void bindDiffUnknownApprovalIsRejected() {
    ChatOutcome.Accepted code = afterDesignApprove();
    Map<String, Object> bind = decision(code.pending(), "APPROVE", "sha256:agent-diff");
    bind.put("approval_id", "APR-00000000-0000-4000-8000-000000000099");
    ChatOutcome outcome = intake.bindDiff(bind);
    assertThat(outcome).isInstanceOf(ChatOutcome.Rejected.class);
    assertThat(engine.require(code.workflowId()).currentState())
        .isEqualTo(WorkflowState.WAITING_CODE_APPROVAL);
  }

  /**
   * 코드 승인은 Local Agent diff 해시 바인딩 없이는 거절된다.
   */
  @Test
  void codeApproveWithoutDiffBindIsRejected() {
    ChatOutcome.Accepted code = afterDesignApprove();
    ChatOutcome outcome = intake.decide(decision(code.pending(), "APPROVE", "sha256:dead"));
    assertThat(outcome).isInstanceOf(ChatOutcome.Rejected.class);
    assertThat(engine.require(code.workflowId()).currentState())
        .isEqualTo(WorkflowState.WAITING_CODE_APPROVAL);
  }

  /**
   * 바인딩된 actual diff 해시로 코드 승인하면 APPLYING_CHANGE로만 가고 apply 도구는 없다.
   */
  @Test
  void codeApproveAfterBindDoesNotApply() {
    ChatOutcome.Accepted code = afterDesignApprove();
    Map<String, Object> bind = decision(code.pending(), "APPROVE", "sha256:agent-diff");
    bind.put("diff_hash", "sha256:agent-diff");
    assertThat(intake.bindDiff(bind)).isInstanceOf(ChatOutcome.Accepted.class);
    ChatOutcome outcome = intake.decide(bind);
    ChatOutcome.Accepted applying = (ChatOutcome.Accepted) outcome;
    assertThat(engine.require(applying.workflowId()).currentState())
        .isEqualTo(WorkflowState.APPLYING_CHANGE);
    assertThat(applying.pending()).isNull();
    assertThat(harness.invokedTools()).doesNotContain("workspace.apply_changes");
  }

  /**
   * Local Agent apply.completed 후 COMPLETED로 닫힌다. 서버는 apply 도구를 부르지 않는다.
   */
  @Test
  void applyCompletedClosesWorkflow() {
    ChatOutcome.Accepted code = afterDesignApprove();
    Map<String, Object> bind = decision(code.pending(), "APPROVE", "sha256:agent-diff");
    bind.put("diff_hash", "sha256:agent-diff");
    assertThat(intake.bindDiff(bind)).isInstanceOf(ChatOutcome.Accepted.class);
    ChatOutcome.Accepted applying = (ChatOutcome.Accepted) intake.decide(bind);
    ChatOutcome outcome =
        intake.applyCompleted(
            Map.of(
                "work_item_id",
                applying.workItemId(),
                "workspace_id",
                "ws-local",
                "change_set_id",
                "CS-1"));
    ChatOutcome.Accepted done = (ChatOutcome.Accepted) outcome;
    assertThat(engine.require(done.workflowId()).currentState()).isEqualTo(WorkflowState.COMPLETED);
    assertThat(done.progress().getFirst().stage()).isEqualTo(WorkflowState.COMPLETED.name());
    assertThat(done.pending()).isNull();
    assertThat(harness.invokedTools()).doesNotContain("workspace.apply_changes");
  }

  /**
   * COMPLETED 재연결은 대기 승인 없이 progress만 준다.
   */
  @Test
  void restoreAfterCompletedHasNoPending() {
    ChatOutcome.Accepted code = afterDesignApprove();
    Map<String, Object> bind = decision(code.pending(), "APPROVE", "sha256:agent-diff");
    bind.put("diff_hash", "sha256:agent-diff");
    intake.bindDiff(bind);
    ChatOutcome.Accepted applying = (ChatOutcome.Accepted) intake.decide(bind);
    intake.applyCompleted(
        Map.of("work_item_id", applying.workItemId(), "workspace_id", "ws-local"));
    ChatOutcome restored = intake.restore(applying.workItemId());
    ChatOutcome.Accepted again = (ChatOutcome.Accepted) restored;
    assertThat(again.pending()).isNull();
    assertThat(again.progress().getFirst().stage()).isEqualTo(WorkflowState.COMPLETED.name());
  }

  /**
   * work_item_id 불일치는 상태를 바꾸지 않는다.
   */
  @Test
  void bindingMismatchIsFailClosed() {
    ChatOutcome.Accepted created = acceptChat();
    Map<String, Object> bad = decision(created.pending(), "APPROVE", null);
    bad.put("work_item_id", "WI-00000000-0000-4000-8000-000000000099");
    ChatOutcome outcome = intake.decide(bad);
    assertThat(outcome).isInstanceOf(ChatOutcome.Rejected.class);
    assertThat(engine.require(created.workflowId()).currentState())
        .isEqualTo(WorkflowState.WAITING_DESIGN_APPROVAL);
  }

  /**
   * session.restore는 대기 설계 승인을 다시 준다.
   */
  @Test
  void restoreReturnsPendingDesign() {
    ChatOutcome.Accepted created = acceptChat();
    ChatOutcome restored = intake.restore(created.workItemId());
    ChatOutcome.Accepted again = (ChatOutcome.Accepted) restored;
    assertThat(again.pending().approvalId()).isEqualTo(created.pending().approvalId());
    assertThat(again.pending().kind()).isEqualTo(ApprovalView.DESIGN);
    assertThat(again.progress().getFirst().stage())
        .isEqualTo(WorkflowState.WAITING_DESIGN_APPROVAL.name());
  }

  private ChatOutcome.Accepted acceptChat() {
    ChatOutcome outcome =
        chat.submit(
            "{\"message\":\"로그인 실패 시 잠가줘\",\"workspace_id\":\"ws-local\",\"ide_context\":{\"current_file\":\"src/AuthService.java\"}}");
    assertThat(outcome).isInstanceOf(ChatOutcome.Accepted.class);
    return (ChatOutcome.Accepted) outcome;
  }

  private ChatOutcome.Accepted afterDesignApprove() {
    return afterDesignApproveFrom(acceptChat());
  }

  /**
   * 이미 만든 설계 대기를 승인하고 코드 대기 결과를 돌려준다.
   *
   * @param created 설계 대기 Accepted
   */
  private ChatOutcome.Accepted afterDesignApproveFrom(ChatOutcome.Accepted created) {
    return (ChatOutcome.Accepted) intake.decide(decision(created.pending(), "APPROVE", null));
  }

  /**
   * 뷰에 있는 바인딩 필드로 결정 맵을 채운다.
   *
   * @param view 대기 승인
   * @param decision APPROVE 또는 REJECT
   * @param diffHash CODE일 때만. DESIGN이면 무시
   */
  private static Map<String, Object> decision(ApprovalView view, String decision, String diffHash) {
    Map<String, Object> data = new HashMap<>();
    data.put("approval_id", view.approvalId());
    data.put("decision", decision);
    data.put("work_item_id", view.workItemId());
    data.put("workspace_id", view.workspaceId());
    data.put("resource_id", view.resourceId());
    if (view.designIdentity() != null) {
      data.put("design_identity", view.designIdentity());
    }
    if (diffHash != null) {
      data.put("diff_hash", diffHash);
    }
    return data;
  }
}
