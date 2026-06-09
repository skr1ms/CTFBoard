package email

import (
	"context"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

type Message = usecase.EmailMessage
type Mailer = usecase.EmailSender

type UserRepository interface {
	GetByID(ctx context.Context, ID uuid.UUID) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	SetVerified(ctx context.Context, userID uuid.UUID) error
	UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error
}

type VerificationTokenRepository interface {
	Create(ctx context.Context, token *domain.VerificationToken) error
	GetByToken(ctx context.Context, token string) (*domain.VerificationToken, error)
	DeleteByUserAndType(ctx context.Context, userID uuid.UUID, tokenType domain.TokenType) error
}

type TransactionManager interface {
	Run(ctx context.Context, fn func(context.Context) error) error
	RunSerializable(ctx context.Context, fn func(context.Context) error) error
}

type APITokenRevoker interface {
	RevokeAllForUser(ctx context.Context, userID uuid.UUID) error
}
