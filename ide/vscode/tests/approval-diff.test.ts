import assert from "node:assert/strict";
import { test } from "node:test";
import { buildDecide, parseDesignReview } from "../approval/parse";
import { parseDiffReview } from "../diff/parse";
import { formatRunResult, parseRunResult } from "../result/parse";

test("design review is separate from diff and cannot apply", () => {
  const review = parseDesignReview({
    kind: "DESIGN",
    approval_id: "APR-1",
    work_item_id: "WI-1",
    workflow_id: "WF-1",
    workspace_id: "ws-local",
    resource_id: "ctx://work-items/WI-1/design/latest",
    design_identity: "sha256:abc",
    design_summary: "lock account after 5 failures",
    can_apply: true,
  });
  assert.equal(review?.kind, "DESIGN");
  assert.equal(review?.canApply, false);
  assert.equal(review?.designSummary.includes("lock"), true);
  assert.equal(
    parseDesignReview({
      kind: "DESIGN",
      approval_id: "APR-1",
      work_item_id: "WI-1",
      resource_id: "r",
      actual_diff: "--- a/x",
    }),
    null,
  );
  const withoutKind = parseDesignReview({
    approval_id: "APR-1",
    work_item_id: "WI-1",
    resource_id: "ctx://work-items/WI-1/design/latest",
    design_summary: "lock account after 5 failures",
  });
  assert.equal(withoutKind?.kind, "DESIGN");
});

test("diff review requires local agent actual diff not proposal json", () => {
  const ok = parseDiffReview({
    kind: "CODE_CHANGE",
    approval_id: "APR-2",
    work_item_id: "WI-1",
    workspace_id: "ws-local",
    resource_id: "cs://x",
    actual_diff: "--- a/src/A.java\n+++ b/src/A.java\n",
    diff_hash: "sha256:fff",
    can_apply: true,
  });
  assert.equal(ok?.canApply, false);
  assert.ok(ok?.actualDiff.startsWith("--- a/"));
  assert.equal(ok?.unchanged, false);
  const same = parseDiffReview({
    kind: "CODE_CHANGE",
    approval_id: "APR-2",
    work_item_id: "WI-1",
    workspace_id: "ws-local",
    resource_id: "cs://x",
    actual_diff: "--- a/src/A.java\n+++ b/src/A.java\n@@\n context\n",
    diff_hash: "sha256:fff",
    unchanged: true,
    message: "제안 내용이 현재 파일과 같습니다.",
  });
  assert.equal(same?.unchanged, true);
  assert.ok((same?.message ?? "").includes("같습니다"));
  assert.equal(
    parseDiffReview({
      kind: "CODE_CHANGE",
      approval_id: "APR-2",
      work_item_id: "WI-1",
      resource_id: "cs://x",
      proposed_changes_json: "{\"diff\":\"model\"}",
      actual_diff: "--- a/x",
    }),
    null,
  );
});

test("rejected design decide payload has no apply", () => {
  const data = buildDecide(
    {
      approvalId: "APR-1",
      workItemId: "WI-1",
      workflowId: "WF-1",
      workspaceId: "ws-local",
      resourceId: "ctx://d",
      kind: "DESIGN",
      designIdentity: "sha256:abc",
      canApply: false,
    },
    "REJECT",
    "no",
  );
  assert.equal(data.decision, "REJECT");
  assert.equal(data.can_apply, false);
  assert.equal("apply" in data, false);
});

test("build and test results format when present", () => {
  const build = parseRunResult("build.result", { kind: "BUILD", status: "SUCCESS", summary: "BUILD SUCCESS" });
  assert.equal(formatRunResult(build!), "BUILD SUCCESS: BUILD SUCCESS");
  assert.equal(parseRunResult("build.result", {}), null);
});
