package request

type CreateHintRequest struct {
	Content    string `json:"content" Validate:"required,hint_content" example:"This is a hint"`
	Cost       int    `json:"cost" Validate:"gte=0" example:"50"`
	OrderIndex int    `json:"order_index" Validate:"gte=0" example:"0"`
}

type UpdateHintRequest struct {
	Content    string `json:"content" Validate:"required,hint_content" example:"Updated hint content"`
	Cost       int    `json:"cost" Validate:"gte=0" example:"100"`
	OrderIndex int    `json:"order_index" Validate:"gte=0" example:"1"`
}
