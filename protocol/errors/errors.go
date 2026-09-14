package errors

import "fmt"

type Category string

const (
	CategoryProtocol   Category = "PROTOCOL"
	CategoryAuth       Category = "AUTH"
	CategoryPermission Category = "PERMISSION"
	CategoryProject    Category = "PROJECT"
	CategoryWorkspace  Category = "WORKSPACE"
	CategoryTool       Category = "TOOL"
	CategoryBuild      Category = "BUILD"
	CategoryTest       Category = "TEST"
	CategoryGit        Category = "GIT"
	CategoryDependency Category = "DEPENDENCY"
	CategorySecurity   Category = "SECURITY"
	CategoryGuide      Category = "GUIDE"
	CategoryContext    Category = "CONTEXT"
	CategoryUser       Category = "USER"
	CategoryTimeout    Category = "TIMEOUT"
	CategorySystem     Category = "SYSTEM"
)

type Code string

const (
	ProtocolVersionUnsupported Code = "PROTOCOL_VERSION_UNSUPPORTED"
	InvalidRequest             Code = "INVALID_REQUEST"
	InvalidArgument            Code = "INVALID_ARGUMENT"

	PermissionDenied Code = "PERMISSION_DENIED"
	ApprovalRequired Code = "APPROVAL_REQUIRED"
	ApprovalRejected Code = "APPROVAL_REJECTED"
	ApprovalInvalid  Code = "APPROVAL_INVALID"

	ProjectNotFound Code = "PROJECT_NOT_FOUND"

	WorkspaceNotFound     Code = "WORKSPACE_NOT_FOUND"
	WorkspaceFileNotFound Code = "WORKSPACE_FILE_NOT_FOUND"
	WorkspaceFileChanged  Code = "WORKSPACE_FILE_CHANGED"
	WorkspaceStale        Code = "WORKSPACE_STALE"
	InvalidPath           Code = "INVALID_PATH"
	PathOutsideWorkspace  Code = "PATH_OUTSIDE_WORKSPACE"
	SymlinkEscape         Code = "SYMLINK_ESCAPE"
	FileTooLarge          Code = "FILE_TOO_LARGE"
	FileHashMismatch      Code = "FILE_HASH_MISMATCH"
	InvalidChangeSet      Code = "INVALID_CHANGE_SET"
	ChangeSetNotFound     Code = "CHANGE_SET_NOT_FOUND"
	DiffMismatch          Code = "DIFF_MISMATCH"
	ApplyFailed           Code = "APPLY_FAILED"

	ToolNotSupported Code = "TOOL_NOT_SUPPORTED"

	BuildToolNotFound Code = "BUILD_TOOL_NOT_FOUND"
	BuildFailed       Code = "BUILD_FAILED"
	TestFailed        Code = "TEST_FAILED"
	ProcessTimeout    Code = "PROCESS_TIMEOUT"
	ProcessCancelled  Code = "PROCESS_CANCELLED"

	GitNotAvailable  Code = "GIT_NOT_AVAILABLE"
	GitConflict      Code = "GIT_CONFLICT"
	GitDirtyWorktree Code = "GIT_DIRTY_WORKTREE"

	DependencyNotFound      Code = "DEPENDENCY_NOT_FOUND"
	DependencyVulnerable    Code = "DEPENDENCY_VULNERABLE"
	SecurityPolicyViolation Code = "SECURITY_POLICY_VIOLATION"

	GuideNotFound     Code = "GUIDE_NOT_FOUND"
	ContextNotFound   Code = "CONTEXT_NOT_FOUND"
	UserInputRequired Code = "USER_INPUT_REQUIRED"

	Timeout       Code = "TIMEOUT"
	InternalError Code = "INTERNAL_ERROR"
)

type meta struct {
	Category  Category
	Retryable bool
}

var catalog = map[Code]meta{
	ProtocolVersionUnsupported: {CategoryProtocol, false},
	InvalidRequest:             {CategoryProtocol, false},
	InvalidArgument:            {CategoryProtocol, false},

	PermissionDenied: {CategoryPermission, false},
	ApprovalRequired: {CategoryPermission, false},
	ApprovalRejected: {CategoryPermission, false},
	ApprovalInvalid:  {CategoryPermission, false},

	ProjectNotFound: {CategoryProject, false},

	WorkspaceNotFound:     {CategoryWorkspace, false},
	WorkspaceFileNotFound: {CategoryWorkspace, false},
	WorkspaceFileChanged:  {CategoryWorkspace, false},
	WorkspaceStale:        {CategoryWorkspace, false},
	InvalidPath:           {CategoryWorkspace, false},
	PathOutsideWorkspace:  {CategoryWorkspace, false},
	SymlinkEscape:         {CategoryWorkspace, false},
	FileTooLarge:          {CategoryWorkspace, false},
	FileHashMismatch:      {CategoryWorkspace, false},
	InvalidChangeSet:      {CategoryWorkspace, false},
	ChangeSetNotFound:     {CategoryWorkspace, false},
	DiffMismatch:          {CategoryWorkspace, false},
	ApplyFailed:           {CategoryWorkspace, false},

	ToolNotSupported: {CategoryTool, false},

	BuildToolNotFound: {CategoryBuild, false},
	BuildFailed:       {CategoryBuild, false},
	TestFailed:        {CategoryTest, false},
	ProcessTimeout:    {CategoryTimeout, true},
	ProcessCancelled:  {CategoryTool, false},

	GitNotAvailable:  {CategoryGit, false},
	GitConflict:      {CategoryGit, false},
	GitDirtyWorktree: {CategoryGit, false},

	DependencyNotFound:      {CategoryDependency, false},
	DependencyVulnerable:    {CategorySecurity, false},
	SecurityPolicyViolation: {CategorySecurity, false},

	GuideNotFound:     {CategoryGuide, false},
	ContextNotFound:   {CategoryContext, false},
	UserInputRequired: {CategoryUser, false},

	Timeout:       {CategoryTimeout, true},
	InternalError: {CategorySystem, false},
}

func Known(code Code) bool {
	_, ok := catalog[code]
	return ok
}

func Codes() []Code {
	out := make([]Code, 0, len(catalog))
	for c := range catalog {
		out = append(out, c)
	}
	return out
}

func CategoryOf(code Code) (Category, bool) {
	m, ok := catalog[code]
	return m.Category, ok
}

func Retryable(code Code) bool {
	return catalog[code].Retryable
}

type Error struct {
	Code      Code           `json:"code"`
	Category  Category       `json:"category"`
	Message   string         `json:"message"`
	Retryable bool           `json:"retryable"`
	Details   map[string]any `json:"details,omitempty"`
}

func New(code Code, message string) (*Error, error) {
	m, ok := catalog[code]
	if !ok {
		return nil, fmt.Errorf("unknown error code %q", code)
	}
	if message == "" {
		message = string(code)
	}
	return &Error{
		Code:      code,
		Category:  m.Category,
		Message:   message,
		Retryable: m.Retryable,
	}, nil
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}
