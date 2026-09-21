package com.oncode.server.api;

import static org.assertj.core.api.Assertions.assertThat;

import com.oncode.server.support.PostgresSpringBootTest;
import com.oncode.server.workflow.WorkflowState;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;

/**
 * 채팅이 work item을 만들고, 소스 첨부를 거절하며, 경로가 없으면 질문하는지 검증한다.
 */
@PostgresSpringBootTest
class ChatIntakeTest {

  @Autowired private ChatIntake intake;

  @Test
  void chatWithPathCreatesWorkItemAndProgress() {
    String json =
        """
        {"message":"로그에 null 처리를 추가해줘.","ide_context":{"current_file":"src/UserService.java","selection":{"start_line":42,"end_line":67}}}
        """;
    ChatOutcome outcome = intake.submit(json);
    assertThat(outcome).isInstanceOf(ChatOutcome.Accepted.class);
    ChatOutcome.Accepted accepted = (ChatOutcome.Accepted) outcome;
    assertThat(accepted.workItemId()).startsWith("WI-");
    assertThat(accepted.progress()).extracting(ProgressEvent::stage)
        .contains(WorkflowState.RECEIVED.name(), WorkflowState.WAITING_DESIGN_APPROVAL.name());
  }

  @Test
  void chatWithSourceBodyIsRejected() {
    String json =
        """
        {"message":"fix","ide_context":{"current_file":"A.java","content":"class A {}"}}
        """;
    ChatOutcome outcome = intake.submit(json);
    assertThat(outcome).isInstanceOf(ChatOutcome.Rejected.class);
    assertThat(((ChatOutcome.Rejected) outcome).code()).isEqualTo("INVALID_REQUEST");
  }

  @Test
  void chatWithoutFileAsksTextQuestion() {
    ChatOutcome outcome = intake.submit("{\"message\":\"무엇을 고칠까요?\"}");
    assertThat(outcome).isInstanceOf(ChatOutcome.NeedQuestion.class);
    Question q = ((ChatOutcome.NeedQuestion) outcome).question();
    assertThat(q.questionType()).isEqualTo(Question.TEXT);
  }

  @Test
  void resumeExistingWorkItem() {
    ChatOutcome first =
        intake.submit(
            "{\"message\":\"설계해줘\",\"ide_context\":{\"current_file\":\"A.java\"}}");
    ChatOutcome.Accepted created = (ChatOutcome.Accepted) first;
    ChatOutcome second =
        intake.submit(
            "{\"message\":\"이어서\",\"work_item_id\":\"" + created.workItemId() + "\"}");
    assertThat(second).isInstanceOf(ChatOutcome.Accepted.class);
    ChatOutcome.Accepted resumed = (ChatOutcome.Accepted) second;
    assertThat(resumed.workItemId()).isEqualTo(created.workItemId());
    assertThat(resumed.progress()).hasSize(1);
    assertThat(resumed.progress().getFirst().stage())
        .isEqualTo(WorkflowState.WAITING_DESIGN_APPROVAL.name());
  }

  @Test
  void questionFactoryCoversThreeTypes() {
    assertThat(IdeContextGuard.supportedQuestionTypes())
        .containsExactly(Question.SINGLE_SELECT, Question.TEXT, Question.CONFIRM);
    assertThat(IdeContextGuard.typedQuestion(Question.SINGLE_SELECT).options()).isNotEmpty();
    assertThat(IdeContextGuard.typedQuestion(Question.CONFIRM).questionType())
        .isEqualTo(Question.CONFIRM);
  }
}
