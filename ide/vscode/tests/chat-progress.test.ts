import assert from "node:assert/strict";
import { test } from "node:test";
import { buildChatSubmit, containsSourceBody, editorContext } from "../chat/context";
import { parseProgress, progressLabel } from "../progress/labels";
import {
  QUESTION_CONFIRM,
  QUESTION_SINGLE_SELECT,
  QUESTION_TEXT,
  buildAnswer,
  parseQuestion,
} from "../question/parse";

test("chat submit carries path and range only", () => {
  const data = buildChatSubmit("add null check", editorContext("UserService.java", 42, 67));
  assert.equal(containsSourceBody(data), false);
  const ctx = data.ide_context as { current_file: string; selection: { start_line: number } };
  assert.equal(ctx.current_file, "UserService.java");
  assert.equal(ctx.selection.start_line, 42);
});

test("progress labels cover waiting design approval", () => {
  assert.ok(progressLabel("WAITING_DESIGN_APPROVAL").includes("설계"));
  assert.ok(progressLabel("COMPLETED").includes("완료"));
  const parsed = parseProgress({
    stage: "DESIGNING",
    status: "RUNNING",
    message: "구현 설계를 작성하고 있습니다.",
  });
  assert.equal(parsed?.stage, "DESIGNING");
});

test("question parser accepts three types", () => {
  const select = parseQuestion({
    question_id: "q-1",
    question_type: QUESTION_SINGLE_SELECT,
    prompt: "choose",
    options: [{ id: "IMPLEMENT", label: "구현" }],
  });
  assert.equal(select?.options[0]?.id, "IMPLEMENT");
  assert.equal(parseQuestion({ question_type: QUESTION_TEXT, prompt: "path?" })?.questionType, QUESTION_TEXT);
  assert.equal(
    parseQuestion({ question_type: QUESTION_CONFIRM, prompt: "go?" })?.questionType,
    QUESTION_CONFIRM,
  );
  assert.deepEqual(buildAnswer("q-1", "src/A.java"), { question_id: "q-1", answer: "src/A.java" });
});
