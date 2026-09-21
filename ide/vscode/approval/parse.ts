/**
 * 승인 바인딩 필드. IDE는 이 값을 그대로 돌려보낸다.
 */
export interface ApprovalBinding {
  approvalId: string;
  workItemId: string;
  workflowId: string;
  workspaceId: string;
  resourceId: string;
  kind: string;
  designIdentity?: string;
  diffHash?: string;
  canApply: boolean;
}

export interface DesignReview extends ApprovalBinding {
  designSummary: string;
}

/**
 * design.review를 정규화한다. actual_diff가 있으면 코드 리뷰이므로 버린다.
 * kind가 비어 있어도 design_summary/design_identity가 있으면 설계로 본다.
 */
export function parseDesignReview(data: Record<string, unknown> | undefined): DesignReview | null {
  const base = parseBinding(data);
  if (!base) {
    return null;
  }
  if (data && "actual_diff" in data) {
    return null;
  }
  const hasDesignFields =
    typeof data?.design_summary === "string" || typeof data?.design_identity === "string";
  if (base.kind === "CODE_CHANGE") {
    return null;
  }
  if (base.kind !== "DESIGN" && !hasDesignFields) {
    return null;
  }
  return {
    ...base,
    kind: "DESIGN",
    designSummary: typeof data?.design_summary === "string" ? data.design_summary : "",
  };
}

/**
 * 승인 결정 IPC data. apply 필드는 넣지 않는다.
 */
export function buildDecide(
  binding: ApprovalBinding,
  decision: "APPROVE" | "REJECT",
  reason?: string,
): Record<string, unknown> {
  const data: Record<string, unknown> = {
    approval_id: binding.approvalId,
    decision,
    work_item_id: binding.workItemId,
    workflow_id: binding.workflowId,
    workspace_id: binding.workspaceId,
    resource_id: binding.resourceId,
    can_apply: false,
  };
  if (binding.designIdentity) {
    data.design_identity = binding.designIdentity;
  }
  if (binding.diffHash) {
    data.diff_hash = binding.diffHash;
  }
  if (reason) {
    data.reason = reason;
  }
  return data;
}

/**
 * 공통 바인딩을 읽는다. can_apply는 항상 false로 본다.
 */
export function parseBinding(data: Record<string, unknown> | undefined): ApprovalBinding | null {
  if (!data) {
    return null;
  }
  const approvalId = str(data.approval_id);
  const workItemId = str(data.work_item_id);
  const resourceId = str(data.resource_id);
  if (!approvalId || !workItemId || !resourceId) {
    return null;
  }
  return {
    approvalId,
    workItemId,
    workflowId: str(data.workflow_id),
    workspaceId: str(data.workspace_id),
    resourceId,
    kind: str(data.kind),
    designIdentity: str(data.design_identity) || undefined,
    diffHash: str(data.diff_hash) || undefined,
    canApply: false,
  };
}

function str(v: unknown): string {
  return typeof v === "string" ? v : "";
}
