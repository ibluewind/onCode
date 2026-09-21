import * as vscode from "vscode";
import { buildDecide, parseDesignReview, type ApprovalBinding } from "../approval/parse";
import { buildChatSubmit, editorContext } from "../chat/context";
import { parseDiffReview } from "../diff/parse";
import { AgentIpcClient } from "../ipc/client";
import { readEndpointFile, resolveStateDir } from "../ipc/endpoint";
import {
  IPC_TYPE_APPLY_RESULT,
  IPC_TYPE_APPROVAL_DECIDE,
  IPC_TYPE_BUILD_RESULT,
  IPC_TYPE_CHAT_SUBMIT,
  IPC_TYPE_DESIGN_REVIEW,
  IPC_TYPE_DIFF_REVIEW,
  IPC_TYPE_ERROR,
  IPC_TYPE_PROGRESS,
  IPC_TYPE_QUESTION_ANSWER,
  IPC_TYPE_QUESTION_ASK,
  IPC_TYPE_REGISTERED,
  IPC_TYPE_SESSION_RESTORE,
  IPC_TYPE_TEST_RESULT,
} from "../ipc/messages";
import { parseProgress, progressLabel } from "../progress/labels";
import {
  QUESTION_CONFIRM,
  QUESTION_SINGLE_SELECT,
  QUESTION_TEXT,
  buildAnswer,
  parseQuestion,
} from "../question/parse";
import { formatRunResult, parseRunResult } from "../result/parse";

const STATE_WORK_ITEM = "oncode.lastWorkItemId";

/** 같은 승인 모달을 두 번 띄우지 않기 위한 키. */
let activeApprovalKey = "";
/** 모달을 한 번에 하나만 띄운다. */
let approvalQueue: Promise<void> = Promise.resolve();

/**
 * VS Code thin client 진입점. Local Agent IPC에만 붙고 워크스페이스 파일 내용은 읽지 않는다.
 *
 * @param context 구독 해제를 등록할 확장 컨텍스트
 */
export function activate(context: vscode.ExtensionContext): void {
  const output = vscode.window.createOutputChannel("onCode");
  const client = new AgentIpcClient({
    extensionVersion: context.extension.packageJSON.version as string,
    log: (line) => output.appendLine(line),
    loadEndpoint: () => {
      const override = vscode.workspace.getConfiguration("oncode").get<string>("agentStateDir");
      return readEndpointFile(resolveStateDir(override));
    },
  });
  client.onMessage((msg) => {
    if (msg.type === IPC_TYPE_REGISTERED) {
      const workItemId = context.workspaceState.get<string>(STATE_WORK_ITEM);
      client.send({
        type: IPC_TYPE_SESSION_RESTORE,
        data: workItemId ? { work_item_id: workItemId } : {},
      });
    }
    if (msg.type === IPC_TYPE_ERROR) {
      const code = typeof msg.data?.code === "string" ? msg.data.code : "ERROR";
      const message = typeof msg.data?.message === "string" ? msg.data.message : "unknown error";
      output.appendLine(`${code}: ${message}`);
    }
    if (msg.type === IPC_TYPE_PROGRESS) {
      const progress = parseProgress(msg.data);
      if (progress) {
        if (progress.stage === "COMPLETED" || progress.stage === "REJECTED") {
          void context.workspaceState.update(STATE_WORK_ITEM, undefined);
        } else {
          rememberWorkItem(context, msg.data);
        }
        output.appendLine(`${progress.stage}: ${progress.message || progressLabel(progress.stage)}`);
      }
    }
    if (msg.type === IPC_TYPE_QUESTION_ASK) {
      void promptQuestion(client, msg.data);
    }
    if (msg.type === IPC_TYPE_DESIGN_REVIEW) {
      rememberWorkItem(context, msg.data);
      output.appendLine("design.review 수신");
      void enqueueApproval(promptKey(msg.data), () => promptDesign(client, output, msg.data));
    }
    if (msg.type === IPC_TYPE_DIFF_REVIEW) {
      rememberWorkItem(context, msg.data);
      output.appendLine("diff.review 수신");
      void enqueueApproval(promptKey(msg.data), () => promptDiff(client, output, msg.data));
    }
    if (msg.type === IPC_TYPE_BUILD_RESULT || msg.type === IPC_TYPE_TEST_RESULT) {
      const result = parseRunResult(msg.type, msg.data);
      if (result) {
        output.appendLine(formatRunResult(result));
      }
    }
    if (msg.type === IPC_TYPE_APPLY_RESULT) {
      const status = typeof msg.data?.status === "string" ? msg.data.status : "UNKNOWN";
      const message = typeof msg.data?.message === "string" ? msg.data.message : "";
      output.appendLine(`apply.result ${status}${message ? `: ${message}` : ""}`);
    }
  });
  context.subscriptions.push(output);
  context.subscriptions.push({ dispose: () => client.dispose() });
  context.subscriptions.push(
    vscode.commands.registerCommand("oncode.reconnect", () => {
      client.reconnect();
    }),
  );
  context.subscriptions.push(
    vscode.commands.registerCommand("oncode.chat.send", async () => {
      output.show(true);
      const message = await vscode.window.showInputBox({
        title: "onCode",
        prompt: "요청을 입력하세요",
      });
      if (!message) {
        return;
      }
      if (!client.isOpen()) {
        output.appendLine("Send Chat 실패: Local Agent IPC 미연결");
        void vscode.window.showWarningMessage(
          "onCode: Local Agent에 연결되어 있지 않습니다. 에이전트를 기동한 뒤 onCode: Reconnect to Local Agent를 실행하세요.",
        );
        return;
      }
      const editor = vscode.window.activeTextEditor;
      if (!editor) {
        output.appendLine("열린 편집기가 없습니다. 서버가 현재 파일을 물을 수 있습니다.");
      }
      const ctx = editorContext(
        editor?.document.fileName,
        editor ? editor.selection.start.line + 1 : undefined,
        editor ? editor.selection.end.line + 1 : undefined,
      );
      activeApprovalKey = "";
      const sent = client.send({
        type: IPC_TYPE_CHAT_SUBMIT,
        data: buildChatSubmit(message, ctx),
      });
      if (!sent) {
        output.appendLine("Send Chat 실패: 소켓이 닫혔습니다");
        void vscode.window.showWarningMessage("onCode: 전송에 실패했습니다. Reconnect 후 다시 시도하세요.");
        return;
      }
      output.appendLine("요청을 보냈습니다. 설계 검토를 기다립니다.");
    }),
  );
  void client.start();
}

/**
 * 재연결 복원용 work item을 확장 상태에만 둔다. 워크스페이스 파일은 쓰지 않는다.
 */
function rememberWorkItem(
  context: vscode.ExtensionContext,
  data: Record<string, unknown> | undefined,
): void {
  const id = data && typeof data.work_item_id === "string" ? data.work_item_id : "";
  if (id) {
    void context.workspaceState.update(STATE_WORK_ITEM, id);
  }
}

/**
 * 같은 approval_id 모달이 이미 떠 있으면 건너뛴다. 복원 이벤트가 창을 쌓지 않게 한다.
 *
 * @param key approval_id 또는 빈 문자열
 * @param run 모달을 띄우는 함수
 */
function enqueueApproval(key: string, run: () => Promise<void>): Promise<void> {
  if (key && key === activeApprovalKey) {
    return Promise.resolve();
  }
  if (key) {
    activeApprovalKey = key;
  }
  approvalQueue = approvalQueue.then(run).catch(() => undefined);
  return approvalQueue;
}

/**
 * 승인 모달 중복 키. approval_id가 있으면 그것을 쓴다.
 *
 * @param data IPC data
 */
function promptKey(data: Record<string, unknown> | undefined): string {
  return data && typeof data.approval_id === "string" ? data.approval_id : "";
}

/**
 * 설계 리뷰 모달. diff/apply 버튼은 없다.
 */
async function promptDesign(
  client: AgentIpcClient,
  output: vscode.OutputChannel,
  data: Record<string, unknown> | undefined,
): Promise<void> {
  const review = parseDesignReview(data);
  if (!review) {
    activeApprovalKey = "";
    output.appendLine("design.review를 표시하지 않음 (필드 부족)");
    return;
  }
  output.appendLine(`설계 검토: ${review.designSummary}`);
  const picked = await vscode.window.showInformationMessage(
    `설계 검토: ${review.designSummary}`,
    { modal: true },
    "Approve",
    "Reject",
  );
  await sendDecision(client, review, picked);
  activeApprovalKey = "";
}

/**
 * Local Agent actual diff만 보여 준다. Apply는 없다.
 */
async function promptDiff(
  client: AgentIpcClient,
  output: vscode.OutputChannel,
  data: Record<string, unknown> | undefined,
): Promise<void> {
  const review = parseDiffReview(data);
  if (!review) {
    activeApprovalKey = "";
    output.appendLine("diff.review를 표시하지 않음 (actual diff 없음)");
    return;
  }
  output.appendLine("=== actual diff (Local Agent) ===");
  output.appendLine(review.actualDiff);
  if (review.unchanged || review.message) {
    output.appendLine(review.message || "제안 내용이 현재 파일과 같습니다.");
  }
  const prompt = review.unchanged
    ? "내용 변경이 없습니다. 그래도 승인할까요? (디스크는 바뀌지 않습니다)"
    : "Local Agent actual diff를 승인할까요?";
  const picked = await vscode.window.showInformationMessage(
    prompt,
    { modal: true },
    "Approve",
    "Reject",
  );
  await sendDecision(client, review, picked);
  activeApprovalKey = "";
}

/**
 * Approve/Reject만 보낸다. Apply 명령은 만들지 않는다.
 */
async function sendDecision(
  client: AgentIpcClient,
  binding: ApprovalBinding,
  picked: string | undefined,
): Promise<void> {
  if (picked !== "Approve" && picked !== "Reject") {
    return;
  }
  client.send({
    type: IPC_TYPE_APPROVAL_DECIDE,
    data: buildDecide(binding, picked === "Approve" ? "APPROVE" : "REJECT"),
  });
}

/**
 * 질문 타입에 따라 QuickPick/InputBox/확인을 띄우고 답을 Local Agent로 보낸다.
 */
async function promptQuestion(
  client: AgentIpcClient,
  data: Record<string, unknown> | undefined,
): Promise<void> {
  const q = parseQuestion(data);
  if (!q) {
    return;
  }
  let answer: string | undefined;
  if (q.questionType === QUESTION_SINGLE_SELECT) {
    const picked = await vscode.window.showQuickPick(
      q.options.map((o) => ({ label: o.label, description: o.id })),
      { title: q.prompt },
    );
    answer = picked?.description;
  } else if (q.questionType === QUESTION_CONFIRM) {
    const picked = await vscode.window.showInformationMessage(q.prompt, "Yes", "No");
    answer = picked === "Yes" ? "yes" : picked === "No" ? "no" : undefined;
  } else if (q.questionType === QUESTION_TEXT) {
    answer = await vscode.window.showInputBox({ title: "onCode", prompt: q.prompt });
  }
  if (!answer) {
    return;
  }
  client.send({
    type: IPC_TYPE_QUESTION_ANSWER,
    data: buildAnswer(q.questionId, answer),
  });
}

/** 확장이 비활성화될 때 추가 정리. 소켓은 activate에서 등록한 dispose가 닫는다. */
export function deactivate(): void {}
