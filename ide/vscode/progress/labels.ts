/**
 * 워크플로 상태를 사용자 문구로 바꾼다. Agent reasoning은 표시하지 않는다.
 *
 * @param stage WorkflowState 이름
 */
export function progressLabel(stage: string): string {
  switch (stage) {
    case "RECEIVED":
      return "요청을 받았습니다.";
    case "CLASSIFYING":
      return "요청을 분류하고 있습니다.";
    case "DISCOVERING_CONTEXT":
      return "관련 맥락을 찾고 있습니다.";
    case "DESIGNING":
      return "구현 설계를 작성하고 있습니다.";
    case "WAITING_DESIGN_APPROVAL":
      return "설계 승인을 기다리고 있습니다.";
    case "WAITING_CODE_APPROVAL":
      return "코드 승인을 기다리고 있습니다.";
    case "APPLYING_CHANGE":
      return "승인된 변경을 적용하는 단계입니다.";
    case "COMPLETED":
      return "변경 적용이 완료되었습니다.";
    case "REJECTED":
      return "요청이 거절되었습니다.";
    default:
      return stage;
  }
}

export interface ProgressView {
  stage: string;
  status: string;
  message: string;
}

/**
 * workflow.progress data를 정규화한다.
 */
export function parseProgress(data: Record<string, unknown> | undefined): ProgressView | null {
  if (!data || typeof data.stage !== "string") {
    return null;
  }
  return {
    stage: data.stage,
    status: typeof data.status === "string" ? data.status : "",
    message: typeof data.message === "string" ? data.message : progressLabel(data.stage),
  };
}
