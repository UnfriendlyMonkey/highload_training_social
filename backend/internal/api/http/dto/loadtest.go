package dto

type LoadTestEventRequest struct {
	RunID string `json:"run_id"`
	Seq   int64  `json:"seq"`
}

type LoadTestEventResponse struct {
	ID int64 `json:"id"`
}

type LoadTestCountResponse struct {
	RunID string `json:"run_id"`
	Count int64  `json:"count"`
}
