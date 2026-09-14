package policy

import (
	protoerr "oncode/protocol/errors"
	"oncode/protocol/ids"
)

// ApprovalValidator is the Phase 1 approval boundary (mock/local is allowed).
type ApprovalValidator interface {
	Validate(approvalID, changeSetID, approvedDiffHash string) error
}

// LocalApproval accepts any syntactically valid approval_id.
// Real server-backed approval lands in a later phase.
type LocalApproval struct{}

func (LocalApproval) Validate(approvalID, changeSetID, approvedDiffHash string) error {
	if approvalID == "" {
		return mustErr(protoerr.ApprovalRequired, "approval_id required")
	}
	if err := ids.Validate(ids.Approval, approvalID); err != nil {
		return mustErr(protoerr.ApprovalInvalid, "invalid approval_id")
	}
	if changeSetID == "" {
		return mustErr(protoerr.InvalidArgument, "change_set_id required")
	}
	if approvedDiffHash == "" {
		return mustErr(protoerr.DiffMismatch, "approved_diff_hash required")
	}
	return nil
}

func mustErr(code protoerr.Code, msg string) error {
	e, err := protoerr.New(code, msg)
	if err != nil {
		return err
	}
	return e
}
