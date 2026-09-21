export interface RunResultView {
  kind: "BUILD" | "TEST";
  status: string;
  summary: string;
}

/**
 * build.result / test.result data를 표시용으로 바꾼다. 없으면 null.
 */
export function parseRunResult(
  kind: string,
  data: Record<string, unknown> | undefined,
): RunResultView | null {
  if (!data) {
    return null;
  }
  const normalized = kind === "test.result" || data.kind === "TEST" ? "TEST" : "BUILD";
  const summary = typeof data.summary === "string" ? data.summary : "";
  if (!summary && typeof data.status !== "string") {
    return null;
  }
  return {
    kind: normalized,
    status: typeof data.status === "string" ? data.status : "UNKNOWN",
    summary,
  };
}

/**
 * Output 한 줄. 결과가 있을 때만 쓴다.
 */
export function formatRunResult(result: RunResultView): string {
  return `${result.kind} ${result.status}: ${result.summary}`;
}
