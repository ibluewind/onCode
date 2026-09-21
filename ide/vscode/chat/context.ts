/**
 * IDE가 서버에 넘기는 편집기 식별 정보. 소스 본문은 넣지 않는다 (SPEC-07 §14).
 */
export interface IdeEditorContext {
  current_file?: string;
  selection?: { start_line: number; end_line: number };
  cursor?: { line: number; column: number };
}

/**
 * 경로와 1-based line range만 담는다. text 인자가 있으면 거부한다.
 *
 * @param path 현재 파일 경로. 비면 생략
 * @param startLine 선택 시작 줄. 1 이상
 * @param endLine 선택 끝 줄. startLine 이상
 */
export function editorContext(
  path?: string,
  startLine?: number,
  endLine?: number,
): IdeEditorContext {
  const ctx: IdeEditorContext = {};
  if (path && path.trim() !== "" && !path.includes("\n")) {
    ctx.current_file = path;
  }
  if (typeof startLine === "number" && typeof endLine === "number") {
    ctx.selection = { start_line: startLine, end_line: endLine };
  }
  return ctx;
}

/**
 * chat.submit data를 만든다. content/source 필드는 넣지 않는다.
 */
export function buildChatSubmit(
  message: string,
  ctx: IdeEditorContext,
  workItemId?: string,
): Record<string, unknown> {
  const data: Record<string, unknown> = { message, ide_context: ctx };
  if (workItemId) {
    data.work_item_id = workItemId;
  }
  return data;
}

const forbidden = ["content", "source", "selected_text", "file_content"];

/**
 * 실수로 본문이 끼었는지 검사한다. 있으면 true.
 */
export function containsSourceBody(data: Record<string, unknown>): boolean {
  const raw = JSON.stringify(data);
  return forbidden.some((k) => raw.includes(`"${k}"`));
}
