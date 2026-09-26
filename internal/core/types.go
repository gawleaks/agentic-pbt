package core

// Config configures a PBT run.
type Config struct {
	SchemaVersion  int      `json:"schemaVersion"`
	ProjectRoot    string   `json:"projectRoot"`
	Inputs         []string `json:"inputs"`
	Python         []string `json:"python"`
	Command        []string `json:"command"`
	TimeoutSeconds int      `json:"timeoutSeconds"`
}

// RequiredProperty describes one property that must be tested.
type RequiredProperty struct {
	ID        string `json:"id"`
	Statement string `json:"statement"`
	Generator string `json:"generator"`
	Oracle    string `json:"oracle"`
	Test      string `json:"test"`
}

// Contract describes either required PBT behavior or why it is not applicable.
type Contract struct {
	SchemaVersion int                `json:"schemaVersion"`
	Task          string             `json:"task"`
	Applicability string             `json:"applicability"`
	Behavior      string             `json:"behavior"`
	Reason        string             `json:"reason"`
	Properties    []RequiredProperty `json:"properties"`
}

type VerificationState string

const (
	VerificationPending       VerificationState = "pending"
	VerificationRunning       VerificationState = "running"
	VerificationPassed        VerificationState = "passed"
	VerificationFailed        VerificationState = "failed"
	VerificationStale         VerificationState = "stale"
	VerificationNotApplicable VerificationState = "not_applicable"
)

// DirtyBaseline records relevant changes that existed when a task began.
type DirtyBaseline struct {
	Modified  []string `json:"modified"`
	Untracked []string `json:"untracked"`
	Deleted   []string `json:"deleted"`
}

// TaskState records one task's local verification lifecycle.
type TaskState struct {
	SchemaVersion      int               `json:"schemaVersion"`
	Task               string            `json:"task"`
	Baseline           DirtyBaseline     `json:"baseline"`
	CurrentFingerprint string            `json:"currentFingerprint"`
	RepairCount        int               `json:"repairCount"`
	LastResultID       string            `json:"lastResultId"`
	VerificationState  VerificationState `json:"verificationState"`
	WorkflowState      WorkflowState     `json:"workflowState"`
	RunDirectory       string            `json:"runDirectory"`
}

type WorkflowState string

const (
	WorkflowActive    WorkflowState = "active"
	WorkflowComplete  WorkflowState = "complete"
	WorkflowBlocked   WorkflowState = "blocked"
	WorkflowCancelled WorkflowState = "cancelled"
)

// Result is the outcome of a PBT operation.
type Result struct {
	SchemaVersion     int               `json:"schemaVersion"`
	Task              string            `json:"task"`
	VerificationState VerificationState `json:"verificationState"`
	WorkflowState     WorkflowState     `json:"workflowState"`
	ReasonCode        string            `json:"reasonCode"`
	NextAction        string            `json:"nextAction"`
	FollowUp          string            `json:"followUp,omitempty"`
}
