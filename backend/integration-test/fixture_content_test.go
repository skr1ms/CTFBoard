package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func (f *TestFixture) NewMinimalBackupData(t *testing.T) *domain.BackupData {
	t.Helper()

	comp, err := f.CompetitionRepo.Get(context.Background())
	require.NoError(t, err)

	return &domain.BackupData{
		Version:     domain.BackupVersion,
		ExportedAt:  time.Now().UTC(),
		Competition: comp,
		Challenges:  []domain.ChallengeExport{},
		Teams:       []domain.TeamExport{},
		Users:       []domain.UserExport{},
		Awards:      []domain.Award{},
		Solves:      []domain.Solve{},
		Files:       []domain.File{},
	}
}

func (f *TestFixture) GetDefaultAppSettings(t *testing.T) *domain.Settings {
	t.Helper()

	ctx := context.Background()
	settings, err := f.SettingsRepo.Get(ctx)
	require.NoError(t, err)

	return settings
}

func (f *TestFixture) CreateTag(t *testing.T, suffix string) *domain.Tag {
	t.Helper()

	unique := suffix + "_" + uuid.NewString()[:8]
	ctx := context.Background()
	tag := &domain.Tag{
		Name:  "tag_" + unique,
		Color: "#ff0000",
	}
	err := f.TagRepo.Create(ctx, tag)
	require.NoError(t, err)
	gotTag, err := f.TagRepo.GetByName(ctx, tag.Name)
	require.NoError(t, err)

	tag.ID = gotTag.ID

	return tag
}

func (f *TestFixture) CreateTopic(t *testing.T, suffix string) *domain.Topic {
	t.Helper()

	unique := suffix + "_" + uuid.NewString()[:8]
	ctx := context.Background()
	topic := &domain.Topic{Name: "topic_" + unique}
	err := f.TopicRepo.Create(ctx, topic)
	require.NoError(t, err)
	gotTopic, err := f.TopicRepo.GetByName(ctx, topic.Name)
	require.NoError(t, err)

	return gotTopic
}

func (f *TestFixture) CreateComment(t *testing.T, userID, challengeID uuid.UUID, content string) *domain.Comment {
	t.Helper()

	ctx := context.Background()
	comment := &domain.Comment{
		UserID:      userID,
		ChallengeID: challengeID,
		Content:     content,
	}
	err := f.CommentRepo.Create(ctx, comment)
	require.NoError(t, err)

	return comment
}

func (f *TestFixture) CreateBracket(t *testing.T, suffix string) *domain.Bracket {
	t.Helper()

	unique := suffix + "_" + uuid.NewString()[:8]
	ctx := context.Background()
	bracket := &domain.Bracket{
		Name:        "bracket_" + unique,
		Description: "desc",
		IsDefault:   false,
	}
	err := f.BracketRepo.Create(ctx, bracket)
	require.NoError(t, err)
	got, err := f.BracketRepo.GetByName(ctx, bracket.Name)
	require.NoError(t, err)

	bracket.ID = got.ID

	return bracket
}

func (f *TestFixture) CreatePage(t *testing.T, suffix string, isDraft bool) *domain.Page {
	t.Helper()

	unique := suffix + "_" + uuid.NewString()[:8]
	ctx := context.Background()
	page := &domain.Page{
		Title:      "Page " + unique,
		Slug:       "page-" + unique,
		Content:    "content",
		IsDraft:    isDraft,
		OrderIndex: 0,
	}
	err := f.PageRepo.Create(ctx, page)
	require.NoError(t, err)
	got, err := f.PageRepo.GetBySlug(ctx, page.Slug)
	require.NoError(t, err)

	page.ID = got.ID

	return page
}

func (f *TestFixture) CreateNotification(t *testing.T, suffix string) *domain.Notification {
	t.Helper()

	unique := suffix + "_" + uuid.NewString()[:8]
	ctx := context.Background()
	notif := &domain.Notification{
		Title:    "Notif " + unique,
		Content:  "content",
		Type:     domain.NotificationInfo,
		IsPinned: false,
		IsGlobal: true,
	}
	err := f.NotificationRepo.Create(ctx, notif)
	require.NoError(t, err)

	return notif
}

func (f *TestFixture) CreateVerificationToken(t *testing.T, userID uuid.UUID, tokenType domain.TokenType) *domain.VerificationToken {
	t.Helper()

	ctx := context.Background()
	tok := &domain.VerificationToken{
		UserID:    userID,
		Token:     "token_" + uuid.New().String(),
		Type:      tokenType,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	err := f.VerificationTokenRepo.Create(ctx, tok)
	require.NoError(t, err)

	return tok
}

func (f *TestFixture) CreateField(t *testing.T, suffix string, entityType domain.EntityType) *domain.Field {
	t.Helper()

	unique := suffix + "_" + uuid.NewString()[:8]
	ctx := context.Background()
	field := &domain.Field{
		Name:       "field_" + unique,
		FieldType:  domain.FieldTypeText,
		EntityType: entityType,
		Required:   false,
		OrderIndex: 0,
	}
	err := f.FieldRepo.Create(ctx, field)
	require.NoError(t, err)

	return field
}

func (f *TestFixture) ResetAppSettings(t *testing.T) {
	t.Helper()

	ctx := context.Background()
	_, err := f.Pool.Exec(ctx, `
		UPDATE app_settings SET
			app_name = 'CTF Platform',
			verify_emails = TRUE,
			frontend_url = 'http://localhost:3000',
			cors_origins = 'http://localhost:3000,http://localhost:5173',
			resend_enabled = FALSE,
			resend_from_email = 'noreply@ctf-platform.local',
			resend_from_name = 'CTF Platform',
			verify_ttl_hours = 24,
			reset_ttl_hours = 1,
			submit_limit_per_user = 10,
			submit_limit_duration_min = 1,
			scoreboard_visible = 'public',
			registration_open = TRUE,
			writeup_enabled = TRUE,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = 1
	`)
	require.NoError(t, err)
}
