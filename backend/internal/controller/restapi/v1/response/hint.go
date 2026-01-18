package response

type HintResponse struct {
	Id         string  `json:"id"`
	Cost       int     `json:"cost"`
	OrderIndex int     `json:"order_index"`
	Content    *string `json:"content,omitempty"`
	Unlocked   bool    `json:"unlocked"`
}

type HintAdminResponse struct {
	Id          string `json:"id"`
	ChallengeId string `json:"challenge_id"`
	Content     string `json:"content"`
	Cost        int    `json:"cost"`
	OrderIndex  int    `json:"order_index"`
}
