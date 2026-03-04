package store

// ProjectStats holds aggregate counts for a project's activity.
type ProjectStats struct {
	SignalCount    int `json:"signal_count"`
	CandidateCount int `json:"candidate_count"`
	GenerationCount int `json:"generation_count"`
	PipelineCount  int `json:"pipeline_count"`
}
