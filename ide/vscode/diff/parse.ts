import { parseBinding, type ApprovalBinding } from "../approval/parse";

export interface DiffReview extends ApprovalBinding {
  actualDiff: string;
  changeSetId: string;
  /** 제안이 현재 파일과 같아 실질 변경이 없을 때 true */
  unchanged: boolean;
  message: string;
}

/**
 * diff.review를 정규화한다. Local Agent actual_diff가 없으면 버린다.
 * proposed_changes_json이 남아 있으면 모델 제안을 그대로 보여주려는 것이므로 거부한다.
 */
export function parseDiffReview(data: Record<string, unknown> | undefined): DiffReview | null {
  const base = parseBinding(data);
  if (!base || (base.kind && base.kind !== "CODE_CHANGE")) {
    return null;
  }
  if (!data || typeof data.actual_diff !== "string" || data.actual_diff.length === 0) {
    return null;
  }
  if ("proposed_changes_json" in data) {
    return null;
  }
  return {
    ...base,
    kind: "CODE_CHANGE",
    actualDiff: data.actual_diff,
    changeSetId: typeof data.change_set_id === "string" ? data.change_set_id : "",
    canApply: false,
    unchanged: data.unchanged === true,
    message: typeof data.message === "string" ? data.message : "",
  };
}
