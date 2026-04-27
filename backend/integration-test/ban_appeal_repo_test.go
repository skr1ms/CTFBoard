package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func createAppealFixture(t *testing.T, f *TestFixture, userID uuid.UUID, createdAgo time.Duration) *domain.BanAppeal {
	t.Helper()

	appeal := &domain.BanAppeal{
		UserID:   userID,
		Message:  "please unban me",
		Decision: domain.AppealDecisionPending,
	}
	err := f.BanAppealRepo.Create(context.Background(), appeal)
	require.NoError(t, err)

	if createdAgo > 0 {
		_, err = f.Pool.Exec(context.Background(),
			"UPDATE ban_appeals SET created_at = $1 WHERE id = $2",
			time.Now().Add(-createdAgo), appeal.ID,
		)
		require.NoError(t, err)

		appeal.CreatedAt = time.Now().Add(-createdAgo)
	}

	return appeal
}

func TestBanAppealRepo_Create_Success(t *testing.T) {
	t.Parallel()
	f := SetupTestFixture(t)
	user := f.CreateUser(t, "appeal_create")
	ctx := context.Background()

	appeal := &domain.BanAppeal{
		UserID:  user.ID,
		Message: "please unban",
	}
	err := f.BanAppealRepo.Create(ctx, appeal)
	require.NoError(t, err)

	assert.NotEqual(t, uuid.Nil, appeal.ID)
	assert.Equal(t, domain.AppealDecisionPending, appeal.Decision)
	assert.False(t, appeal.CreatedAt.IsZero())
}

func TestBanAppealRepo_GetByID_Success(t *testing.T) {
	t.Parallel()
	f := SetupTestFixture(t)
	user := f.CreateUser(t, "appeal_getbyid")

	created := createAppealFixture(t, f, user.ID, 0)

	got, err := f.BanAppealRepo.GetByID(context.Background(), created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, got.ID)
	assert.Equal(t, user.ID, got.UserID)
	assert.Equal(t, "please unban me", got.Message)
	assert.Equal(t, domain.AppealDecisionPending, got.Decision)
	assert.Nil(t, got.ReviewedAt)
	assert.Nil(t, got.AdminResponse)
}

func TestBanAppealRepo_GetByID_NotFound(t *testing.T) {
	t.Parallel()
	f := SetupTestFixture(t)

	_, err := f.BanAppealRepo.GetByID(context.Background(), uuid.New())
	assert.ErrorIs(t, err, apperr.ErrAppealNotFound)
}

func TestBanAppealRepo_GetByUserID_ReturnsUserAppeals(t *testing.T) {
	t.Parallel()
	f := SetupTestFixture(t)
	user1 := f.CreateUser(t, "appeal_getuid1")
	user2 := f.CreateUser(t, "appeal_getuid2")

	createAppealFixture(t, f, user1.ID, 10*24*time.Hour)
	createAppealFixture(t, f, user1.ID, 5*24*time.Hour)
	createAppealFixture(t, f, user2.ID, 0)

	got, err := f.BanAppealRepo.GetByUserID(context.Background(), user1.ID)
	require.NoError(t, err)
	require.Len(t, got, 2)

	for _, a := range got {
		assert.Equal(t, user1.ID, a.UserID)
	}
}

func TestBanAppealRepo_GetByUserID_Empty(t *testing.T) {
	t.Parallel()
	f := SetupTestFixture(t)

	got, err := f.BanAppealRepo.GetByUserID(context.Background(), uuid.New())
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestBanAppealRepo_GetLatestByUserID_Success(t *testing.T) {
	t.Parallel()
	f := SetupTestFixture(t)
	user := f.CreateUser(t, "appeal_latest")

	older := createAppealFixture(t, f, user.ID, 8*24*time.Hour)
	newer := createAppealFixture(t, f, user.ID, 1*time.Hour)

	got, err := f.BanAppealRepo.GetLatestByUserID(context.Background(), user.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, newer.ID, got.ID)

	_ = older
}

func TestBanAppealRepo_GetLatestByUserID_NotFound(t *testing.T) {
	t.Parallel()
	f := SetupTestFixture(t)

	got, err := f.BanAppealRepo.GetLatestByUserID(context.Background(), uuid.New())
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestBanAppealRepo_List_NoFilter_Pagination(t *testing.T) {
	t.Parallel()
	f := SetupTestFixture(t)

	// Create 5 appeals from different users
	for range 5 {
		u := f.CreateUser(t, "appeal_list_nof")
		createAppealFixture(t, f, u.ID, 0)
	}

	ctx := context.Background()

	// page 1, limit=2, offset=0 -> 2 items
	page1, total, err := f.BanAppealRepo.List(ctx, nil, 2, 0)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, total, int64(5))
	assert.Len(t, page1, 2)

	// page 2, limit=2, offset=2 -> 2 items
	page2, _, err := f.BanAppealRepo.List(ctx, nil, 2, 2)
	require.NoError(t, err)
	assert.Len(t, page2, 2)

	// IDs on page1 and page2 must not overlap
	p1IDs := make(map[uuid.UUID]bool)

	for _, a := range page1 {
		p1IDs[a.ID] = true
	}

	for _, a := range page2 {
		assert.False(t, p1IDs[a.ID], "pages must not overlap")
	}
}

func TestBanAppealRepo_List_FilterByDecision_Pending(t *testing.T) {
	t.Parallel()
	f := SetupTestFixture(t)
	ctx := context.Background()

	u1 := f.CreateUser(t, "appeal_flt_pend1")
	u2 := f.CreateUser(t, "appeal_flt_pend2")
	u3 := f.CreateUser(t, "appeal_flt_pend3")

	pendingAppeal := createAppealFixture(t, f, u1.ID, 0)

	// Resolve appeal for u2
	resolved := createAppealFixture(t, f, u2.ID, 2*time.Hour)
	resolved.Decision = domain.AppealDecisionResolved
	require.NoError(t, f.BanAppealRepo.Update(ctx, resolved))

	// Reject appeal for u3
	rejected := createAppealFixture(t, f, u3.ID, 3*time.Hour)
	rejected.Decision = domain.AppealDecisionRejected
	require.NoError(t, f.BanAppealRepo.Update(ctx, rejected))

	dec := domain.AppealDecisionPending
	got, total, err := f.BanAppealRepo.List(ctx, &dec, 100, 0)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, total, int64(1))

	for _, a := range got {
		assert.Equal(t, domain.AppealDecisionPending, a.Decision)
	}

	// Verify pending appeal is present in results
	found := false

	for _, a := range got {
		if a.ID == pendingAppeal.ID {
			found = true

			break
		}
	}

	assert.True(t, found, "pending appeal must be in filtered results")
}

func TestBanAppealRepo_List_FilterByDecision_Resolved(t *testing.T) {
	t.Parallel()
	f := SetupTestFixture(t)
	ctx := context.Background()

	u := f.CreateUser(t, "appeal_flt_res")
	appeal := createAppealFixture(t, f, u.ID, 2*time.Hour)
	appeal.Decision = domain.AppealDecisionResolved
	require.NoError(t, f.BanAppealRepo.Update(ctx, appeal))

	dec := domain.AppealDecisionResolved
	got, total, err := f.BanAppealRepo.List(ctx, &dec, 100, 0)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, total, int64(1))

	for _, a := range got {
		assert.Equal(t, domain.AppealDecisionResolved, a.Decision)
	}
}

func TestBanAppealRepo_Update_SetsReviewedAt(t *testing.T) {
	t.Parallel()
	f := SetupTestFixture(t)
	user := f.CreateUser(t, "appeal_upd_reviewed")

	appeal := createAppealFixture(t, f, user.ID, 2*time.Hour)
	assert.Nil(t, appeal.ReviewedAt)

	appeal.Decision = domain.AppealDecisionRejected
	err := f.BanAppealRepo.Update(context.Background(), appeal)
	require.NoError(t, err)

	require.NotNil(t, appeal.ReviewedAt)
	assert.WithinDuration(t, time.Now(), *appeal.ReviewedAt, 5*time.Second)

	// Verify persisted
	got, err := f.BanAppealRepo.GetByID(context.Background(), appeal.ID)
	require.NoError(t, err)
	require.NotNil(t, got.ReviewedAt)
	assert.Equal(t, domain.AppealDecisionRejected, got.Decision)
}

func TestBanAppealRepo_Update_AdminResponse(t *testing.T) {
	t.Parallel()
	f := SetupTestFixture(t)
	user := f.CreateUser(t, "appeal_upd_resp")

	appeal := createAppealFixture(t, f, user.ID, 1*time.Hour)

	resp := "your appeal has been reviewed"
	appeal.Decision = domain.AppealDecisionResolved
	appeal.AdminResponse = &resp
	err := f.BanAppealRepo.Update(context.Background(), appeal)
	require.NoError(t, err)

	got, err := f.BanAppealRepo.GetByID(context.Background(), appeal.ID)
	require.NoError(t, err)
	require.NotNil(t, got.AdminResponse)
	assert.Equal(t, resp, *got.AdminResponse)
}

func TestBanAppealRepo_Update_NotFound(t *testing.T) {
	t.Parallel()
	f := SetupTestFixture(t)

	nonExistent := &domain.BanAppeal{
		ID:       uuid.New(),
		Decision: domain.AppealDecisionRejected,
	}
	err := f.BanAppealRepo.Update(context.Background(), nonExistent)
	assert.ErrorIs(t, err, apperr.ErrAppealNotFound)
}
