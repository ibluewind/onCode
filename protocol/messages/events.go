package messages

const (
	EventAgentRegister    = "agent.register"
	EventAgentHeartbeat   = "agent.heartbeat"
	EventChatSubmit       = "chat.submit"
	EventWorkflowProgress = "workflow.progress"
	EventQuestionAsk      = "question.ask"
	EventQuestionAnswer   = "question.answer"
	EventDesignReview     = "design.review"
	EventChangeProposed   = "change.proposed"
	EventDiffReview       = "diff.review"
	EventApprovalDecide   = "approval.decide"
	EventApprovalBind     = "approval.bind"
	EventApplyCompleted   = "apply.completed"
	EventSessionRestore   = "session.restore"
	EventBuildResult      = "build.result"
	EventTestResult       = "test.result"
	EventError            = "error"
)
