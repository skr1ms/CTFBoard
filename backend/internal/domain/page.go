package domain

import (
	"time"

	"github.com/google/uuid"
)

// Page is a static content page authored by admins, addressable by slug.
type Page struct {
	ID         uuid.UUID `json:"id"`
	Title      string    `json:"title"`
	Slug       string    `json:"slug"`
	Content    string    `json:"content"`
	IsDraft    bool      `json:"is_draft"`
	OrderIndex int       `json:"order_index"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// PageListItem is a lightweight projection of Page used in navigation and list endpoints.
type PageListItem struct {
	ID         uuid.UUID `json:"id"`
	Title      string    `json:"title"`
	Slug       string    `json:"slug"`
	OrderIndex int       `json:"order_index"`
}
