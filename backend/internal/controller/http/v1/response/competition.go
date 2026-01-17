package response

type CompetitionResponse struct {
	Id         int     `json:"id"`
	Name       string  `json:"name"`
	StartTime  *string `json:"start_time"`
	EndTime    *string `json:"end_time"`
	FreezeTime *string `json:"freeze_time"`
	IsPaused   bool    `json:"is_paused"`
	IsPublic   bool    `json:"is_public"`
	Status     string  `json:"status"`
}

type CompetitionStatusResponse struct {
	Status            string  `json:"status"`
	Name              string  `json:"name"`
	StartTime         *string `json:"start_time"`
	EndTime           *string `json:"end_time"`
	SubmissionAllowed bool    `json:"submission_allowed"`
}
