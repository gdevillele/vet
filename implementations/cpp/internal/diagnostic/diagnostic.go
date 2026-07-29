package diagnostic

type Severity string

const SeverityError Severity = "error"

type Diagnostic struct {
	RuleID   string   `json:"rule_id"`
	Severity Severity `json:"severity"`
	Message  string   `json:"message"`
	File     string   `json:"file"`
	Line     int      `json:"line"`
	Column   int      `json:"column"`
}
