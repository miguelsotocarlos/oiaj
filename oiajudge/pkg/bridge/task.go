package bridge

type Task struct {
	Id               Id       `json:"id"`
	Name             string   `json:"name"`
	Title            string   `json:"title"`
	Tags             []string `json:"tags"`
	Statement        []byte   `json:"-"`
	Deleted          bool     `json:"-"`
	MaxScore         float64  `json:"max_score"`
	Multiplier       float64  `json:"multiplier"`
	SubmissionFormat []string `json:"submission_format"`
}
