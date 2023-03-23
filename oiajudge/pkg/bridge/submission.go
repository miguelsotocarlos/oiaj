package bridge

import "time"

type ExecutionTime = float64
type MemoryUsage = int64

type Score struct {
	Score    float64 `json:"score"`
	MaxScore float64 `json:"max_score"`
}

type TestcaseResult struct {
	Testcase      string        `json:"testcase"`
	Score         Score         `json:"score"`
	ExecutionTime ExecutionTime `json:"execution_time"`
	MemoryUsage   MemoryUsage   `json:"memory_usage"`

	// Some checkers produce custom messages when scoring a testcase.
	// Most of the time Message = "Output is correct"
	Message string `json:"message"`
}

type SubtaskResult struct {
	Subtask   int64    `json:"subtask"`
	Score     Score    `json:"score"`
	Testcases []string `json:"testcases"`
}

type SubmissionResult struct {
	Score     Score            `json:"score"`
	Subtasks  []SubtaskResult  `json:"subtasks"`
	Testcases []TestcaseResult `json:"testcases"`
}

type SubmissionStatus string

const (
	COMPILING          SubmissionStatus = "compiling"
	COMPILATION_FAILED SubmissionStatus = "compilation_failed"
	EVALUATING         SubmissionStatus = "evaluating"
	SCORING            SubmissionStatus = "scoring"
	SCORED             SubmissionStatus = "scored"
)

type Submission struct {
	Id                 Id                `json:"id"`
	UserId             Id                `json:"user_id"`
	ProblemId          Id                `json:"problem_id"`
	SubmissionStatus   SubmissionStatus  `json:"submission_status"`
	Timestamp          time.Time         `json:"timestamp"`
	CompilationMessage string            `json:"compilation_message"`
	Result             *SubmissionResult `json:"result"`
	Deleted            bool              `json:"-"`
}
