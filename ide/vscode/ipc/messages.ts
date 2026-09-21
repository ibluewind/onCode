/** IDE Bridge JSON 프레임. 와이어 필드는 snake_case다. */
export interface IpcMessage {
  type: string;
  message_id?: string;
  data?: Record<string, unknown>;
}

export const IPC_TYPE_REGISTER = "ide.register";
export const IPC_TYPE_REGISTERED = "ide.registered";
export const IPC_TYPE_ERROR = "error";
export const IPC_TYPE_CHAT_SUBMIT = "chat.submit";
export const IPC_TYPE_CHAT_ACCEPTED = "chat.accepted";
export const IPC_TYPE_PROGRESS = "workflow.progress";
export const IPC_TYPE_QUESTION_ASK = "question.ask";
export const IPC_TYPE_QUESTION_ANSWER = "question.answer";
export const IPC_TYPE_DESIGN_REVIEW = "design.review";
export const IPC_TYPE_DIFF_REVIEW = "diff.review";
export const IPC_TYPE_APPROVAL_DECIDE = "approval.decide";
export const IPC_TYPE_SESSION_RESTORE = "session.restore";
export const IPC_TYPE_BUILD_RESULT = "build.result";
export const IPC_TYPE_TEST_RESULT = "test.result";
/** Local Agent가 승인 후 디스크에 반영한 결과. 확장은 표시만 한다. */
export const IPC_TYPE_APPLY_RESULT = "apply.result";
