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
	SchemaVersion  int                `json:"schemaVersion"`
	Task           string             `json:"task"`
	Applicability  string             `json:"applicability"`
	Behavior       string             `json:"behavior"`
	Reason         string             `json:"reason"`
	Properties     []RequiredProperty `json:"properties"`
}

// Result is the outcome of a PBT operation.
type Result struct {
	Success bool `json:"success"`
}
