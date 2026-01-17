package request

type CreateHintRequest struct {
	Content    string `json:"content" validate:"required" example:"This is a hint"`
	Cost       int    `json:"cost" validate:"gte=0" example:"50"`
	OrderIndex int    `json:"order_index" example:"0"`
}

type UpdateHintRequest struct {
	Content    string `json:"content" validate:"required" example:"Updated hint content"`
	Cost       int    `json:"cost" validate:"gte=0" example:"100"`
	OrderIndex int    `json:"order_index" example:"1"`
}
