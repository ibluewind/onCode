export const QUESTION_SINGLE_SELECT = "SINGLE_SELECT";
export const QUESTION_TEXT = "TEXT";
export const QUESTION_CONFIRM = "CONFIRM";

export interface QuestionOption {
  id: string;
  label: string;
}

export interface QuestionAsk {
  questionId: string;
  questionType: string;
  prompt: string;
  options: QuestionOption[];
}

/**
 * question.ask data를 세 타입 중 하나로 정규화한다. 모르면 null.
 */
export function parseQuestion(data: Record<string, unknown> | undefined): QuestionAsk | null {
  if (!data || typeof data.prompt !== "string") {
    return null;
  }
  const questionType = String(data.question_type ?? data.questionType ?? "");
  if (
    questionType !== QUESTION_SINGLE_SELECT &&
    questionType !== QUESTION_TEXT &&
    questionType !== QUESTION_CONFIRM
  ) {
    return null;
  }
  const options: QuestionOption[] = [];
  if (Array.isArray(data.options)) {
    for (const raw of data.options) {
      if (raw && typeof raw === "object") {
        const rec = raw as Record<string, unknown>;
        if (typeof rec.id === "string" && typeof rec.label === "string") {
          options.push({ id: rec.id, label: rec.label });
        }
      }
    }
  }
  return {
    questionId: String(data.question_id ?? ""),
    questionType,
    prompt: data.prompt,
    options,
  };
}

/**
 * 사용자 답을 question.answer data로 만든다.
 */
export function buildAnswer(questionId: string, answer: string): Record<string, unknown> {
  return { question_id: questionId, answer };
}
