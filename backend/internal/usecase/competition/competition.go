package competition

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/wahrwelt-kit/go-cachekit"
	"github.com/wahrwelt-kit/go-logkit"
	"golang.org/x/sync/singleflight"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/txctx"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/cacheutil"
)

const (
	localCompTTL             = 5 * time.Second
	redisCacheTTL            = 15 * time.Second
	boundaryInvalidateWindow = 25 * time.Second

	maxCompetitionNameLen = 200
	maxFlagRegexLen       = 500
)

type CompetitionUseCase struct {
	deps        CompetitionDeps
	sf          singleflight.Group // zero value is valid, no explicit init required
	localComp   atomic.Pointer[domain.Competition]
	localCompAt atomic.Int64 // unix nano of last store
}

type StatisticsCacheInvalidator interface {
	InvalidateStatistics(ctx context.Context) error
}

type CompetitionDeps struct {
	CompetitionRepo       repo.CompetitionRepository
	AuditLogRepo          repo.AuditLogRepository
	TM                    repo.TransactionManager
	Redis                 cachekit.KeyValueStore
	StatsCacheInvalidator StatisticsCacheInvalidator
	ScoreboardCache       cacheutil.ScoreboardCacheInvalidator
	Logger                logkit.Logger
}

var _ usecase.CompetitionUseCase = (*CompetitionUseCase)(nil)

func NewCompetitionUseCase(deps CompetitionDeps) *CompetitionUseCase {
	if deps.Logger == nil {
		deps.Logger = logkit.Noop()
	}

	return &CompetitionUseCase{deps: deps}
}

func timePtrEqual(a, b *time.Time) bool {
	if a == nil && b == nil {
		return true
	}

	if a == nil || b == nil {
		return false
	}

	ta, tb := a.Truncate(time.Second), b.Truncate(time.Second)

	return ta.Equal(tb)
}

// Update transactionally updates the competition settings. It merges partial
// updates from optionals into the current persisted state, validates field
// constraints and time ordering, enforces that mode/start-time/team-size
// settings cannot change while a competition is active or frozen, and applies
// pause/unpause transitions (shifting EndTime and FreezeTime forward by the
// elapsed pause duration on unpause). After the transaction commits, all cache
// layers (local atomic store, Redis, singleflight key) are flushed and an
// audit log entry is written.
func (uc *CompetitionUseCase) Update(ctx context.Context, comp *domain.Competition, optionals *usecase.CompetitionUpdateOptionals, actorID uuid.UUID, clientIP string) error {
	auditLog := &domain.AuditLog{
		UserID:     &actorID,
		Action:     domain.AuditActionUpdate,
		EntityType: domain.AuditEntityCompetition,
		EntityID:   "settings",
		IP:         clientIP,
		Details: map[string]any{
			"message": "competition settings updated",
		},
	}
	now := time.Now()

	err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		current, err := uc.deps.CompetitionRepo.GetForUpdate(ctx)
		if err != nil {
			return fmt.Errorf("CompetitionUseCase - Update - CompetitionRepo.GetForUpdate: %w", err)
		}

		status := current.GetStatusAt(now)
		uc.mergeDefaults(comp, current, optionals)

		if err := uc.validateCompetitionFields(comp); err != nil {
			return err
		}

		if err := uc.validateCompetitionTimes(comp); err != nil {
			return err
		}

		if comp.Mode != "" && !comp.Mode.IsValid() {
			return apperr.NewValidationErrorf("invalid competition mode %q: must be solo_only or teams_only", comp.Mode)
		}

		if err := uc.validateActiveConstraints(comp, current, status); err != nil {
			return err
		}

		uc.applyPauseTransition(comp, current, now)

		if err := uc.validateCompetitionTimesAfterUnpause(comp, current); err != nil {
			return err
		}

		if err := uc.deps.CompetitionRepo.Update(ctx, comp); err != nil {
			return fmt.Errorf("CompetitionUseCase - Update - CompetitionRepo.Update: %w", err)
		}

		if err := uc.deps.AuditLogRepo.Create(ctx, auditLog); err != nil {
			return fmt.Errorf("CompetitionUseCase - Update - AuditLogRepo.Create: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("CompetitionUseCase - Update - TM.Run: %w", err)
	}

	txctx.AfterCommitOrNow(ctx, func(ctx context.Context) {
		uc.localComp.Store(nil)
		uc.localCompAt.Store(0)

		if uc.deps.Redis != nil {
			_ = uc.deps.Redis.Del(ctx, cacheutil.KeyCompetitionGuard)

			err := uc.deps.Redis.Del(ctx, cacheutil.KeyCompetition)
			if err != nil {
				uc.deps.Logger.WithError(err).Warn("CompetitionUseCase - Update: failed to invalidate cache; stale data for up to 5s")
			}
		}

		uc.sf.Forget(cacheutil.KeyCompetition)

		if uc.deps.StatsCacheInvalidator != nil {
			err := uc.deps.StatsCacheInvalidator.InvalidateStatistics(ctx)
			if err != nil {
				uc.deps.Logger.WithError(err).Warn("CompetitionUseCase - Update: failed to invalidate statistics cache")
			}
		}

		if uc.deps.ScoreboardCache != nil {
			uc.deps.ScoreboardCache.InvalidateAll(ctx)
		}
	})

	return nil
}

// mergeDefaults fills nil or zero fields in comp from the currently persisted
// state so that a partial update does not silently erase unchanged settings.
// Nullable time fields (EndTime, FreezeTime) can be explicitly cleared via the
// ClearEndTime/ClearFreezeTime flags in optionals; otherwise they are inherited.
func (uc *CompetitionUseCase) mergeDefaults(comp, current *domain.Competition, optionals *usecase.CompetitionUpdateOptionals) {
	if comp.StartTime == nil {
		comp.StartTime = current.StartTime
	}

	if optionals != nil && optionals.ClearEndTime != nil && *optionals.ClearEndTime {
		comp.EndTime = nil
	} else if comp.EndTime == nil {
		comp.EndTime = current.EndTime
	}

	if optionals != nil && optionals.ClearFreezeTime != nil && *optionals.ClearFreezeTime {
		comp.FreezeTime = nil
	} else if comp.FreezeTime == nil {
		comp.FreezeTime = current.FreezeTime
	}

	if comp.Mode == "" {
		comp.Mode = current.Mode
	}

	if optionals != nil && optionals.MinTeamSize != nil {
		comp.MinTeamSize = *optionals.MinTeamSize
	} else {
		comp.MinTeamSize = current.MinTeamSize
	}

	if optionals != nil && optionals.MaxTeamSize != nil {
		comp.MaxTeamSize = *optionals.MaxTeamSize
	} else {
		comp.MaxTeamSize = current.MaxTeamSize
	}

	if optionals != nil {
		comp.IsPaused = lo.FromPtrOr(optionals.IsPaused, current.IsPaused)
		comp.IsPublic = lo.FromPtrOr(optionals.IsPublic, current.IsPublic)
		comp.AllowTeamSwitch = lo.FromPtrOr(optionals.AllowTeamSwitch, current.AllowTeamSwitch)
		comp.KeepScoreboardFrozenAfterEnd = lo.FromPtrOr(optionals.KeepScoreboardFrozenAfterEnd, current.KeepScoreboardFrozenAfterEnd)
	} else {
		comp.IsPaused = current.IsPaused
		comp.IsPublic = current.IsPublic
		comp.AllowTeamSwitch = current.AllowTeamSwitch
		comp.KeepScoreboardFrozenAfterEnd = current.KeepScoreboardFrozenAfterEnd
	}
}

// validateCompetitionFields validates scalar competition fields: name length,
// optional flag_regex length, positive team sizes, and the constraint that
// min_team_size must not exceed max_team_size when both are set.
func (uc *CompetitionUseCase) validateCompetitionFields(comp *domain.Competition) error {
	if len(comp.Name) == 0 || len(comp.Name) > maxCompetitionNameLen {
		return apperr.NewValidationErrorf("competition name must be 1-200 characters")
	}

	if comp.FlagRegex != nil && len(*comp.FlagRegex) > maxFlagRegexLen {
		return apperr.NewValidationErrorf("flag_regex must be at most 500 characters")
	}

	if comp.MinTeamSize < 1 {
		return apperr.NewValidationErrorf("min_team_size must be >= 1")
	}

	if comp.MaxTeamSize < 1 {
		return apperr.NewValidationErrorf("max_team_size must be >= 1")
	}

	if comp.MinTeamSize > comp.MaxTeamSize {
		return apperr.NewValidationErrorf("min_team_size must be <= max_team_size")
	}

	return nil
}

func (uc *CompetitionUseCase) validateCompetitionTimes(comp *domain.Competition) error {
	if comp.EndTime != nil && comp.StartTime != nil && comp.EndTime.Before(*comp.StartTime) {
		return apperr.NewValidationErrorf("end_time must be after start_time")
	}

	if comp.FreezeTime != nil && comp.EndTime != nil && !comp.FreezeTime.Before(*comp.EndTime) {
		return apperr.NewValidationErrorf("freeze_time must be before end_time")
	}

	if comp.FreezeTime != nil && comp.StartTime != nil && comp.FreezeTime.Before(*comp.StartTime) {
		return apperr.NewValidationErrorf("freeze_time must be after start_time")
	}

	return nil
}

// validateCompetitionTimesAfterUnpause re-validates time ordering after applyPauseTransition
// has shifted EndTime/FreezeTime forward. It returns an actionable error message when the
// shifted freeze_time would end up after end_time so the admin can correct the values.
func (uc *CompetitionUseCase) validateCompetitionTimesAfterUnpause(comp, current *domain.Competition) error {
	if comp.EndTime != nil && comp.StartTime != nil && comp.EndTime.Before(*comp.StartTime) {
		return apperr.NewValidationErrorf("end_time must be after start_time")
	}

	if comp.FreezeTime != nil && comp.EndTime != nil && !comp.FreezeTime.Before(*comp.EndTime) {
		if current.IsPaused && current.PausedAt != nil && comp.FreezeTime != nil &&
			current.FreezeTime != nil && current.PausedAt.Before(*current.FreezeTime) {
			return apperr.NewValidationErrorf("unpausing shifts freeze_time by the pause duration; result would be after end_time - set a later end_time or clear freeze_time")
		}

		return apperr.NewValidationErrorf("freeze_time must be before end_time")
	}

	if comp.FreezeTime != nil && comp.StartTime != nil && comp.FreezeTime.Before(*comp.StartTime) {
		return apperr.NewValidationErrorf("freeze_time must be after start_time")
	}

	return nil
}

// validateActiveConstraints rejects changes to immutable fields (mode, start_time,
// team sizes, allow_team_switch) while the competition is active, frozen, or paused,
// since modifying them mid-competition would corrupt existing submission records.
func (uc *CompetitionUseCase) validateActiveConstraints(comp, current *domain.Competition, status domain.CompetitionStatus) error {
	if status != domain.CompetitionStatusActive && status != domain.CompetitionStatusFrozen && status != domain.CompetitionStatusPaused {
		return nil
	}

	if comp.Mode != current.Mode || !timePtrEqual(comp.StartTime, current.StartTime) {
		return apperr.ErrCompetitionActiveCannotUpdate
	}

	if comp.MinTeamSize != current.MinTeamSize || comp.MaxTeamSize != current.MaxTeamSize {
		return apperr.ErrCompetitionActiveCannotUpdate
	}

	if comp.AllowTeamSwitch != current.AllowTeamSwitch {
		return apperr.ErrCompetitionActiveCannotUpdate
	}

	return nil
}

// applyPauseTransition mutates comp in place to apply a pause or unpause
// transition. On a pause (was running, now paused) it records now as PausedAt
// On an unpause (was paused, now running) it computes the elapsed pause
// duration from PausedAt to now (clamped to StartTime if PausedAt predates
// it) and shifts EndTime and FreezeTime forward by that duration, but only
// when those fields have not been changed by the caller in this request. If
// PausedAt was nil on an already-paused competition, now is used as a fallback
// and a warning is logged.
func (uc *CompetitionUseCase) applyPauseTransition(comp, current *domain.Competition, now time.Time) {
	wasPaused := current.IsPaused
	isPausing := comp.IsPaused

	if !wasPaused && isPausing {
		comp.PausedAt = &now

		return
	}

	if wasPaused && !isPausing {
		pausedAt := current.PausedAt
		if pausedAt == nil {
			uc.deps.Logger.Warn("CompetitionUseCase - applyPauseTransition: competition was paused but paused_at was nil; using now as fallback")

			pausedAt = &now
		}

		if comp.EndTime != nil && !pausedAt.Before(*comp.EndTime) {
			comp.PausedAt = nil

			return
		}

		effectivePausedAt := pausedAt
		if comp.StartTime != nil && pausedAt.Before(*comp.StartTime) {
			effectivePausedAt = comp.StartTime
		}

		pauseDuration := now.Sub(*effectivePausedAt)
		if pauseDuration <= 0 {
			comp.PausedAt = nil

			return
		}

		endTimeUnchanged := comp.EndTime != nil && timePtrEqual(comp.EndTime, current.EndTime)
		if endTimeUnchanged {
			shifted := comp.EndTime.Add(pauseDuration)
			comp.EndTime = &shifted
		}

		freezeTimeUnchanged := comp.FreezeTime != nil && timePtrEqual(comp.FreezeTime, current.FreezeTime)
		if freezeTimeUnchanged && pausedAt.Before(*comp.FreezeTime) {
			shifted := comp.FreezeTime.Add(pauseDuration)
			comp.FreezeTime = &shifted
		}

		comp.PausedAt = nil

		return
	}

	if !isPausing {
		comp.PausedAt = nil
	} else {
		comp.PausedAt = current.PausedAt
	}
}

func (uc *CompetitionUseCase) GetStatus(ctx context.Context) (domain.CompetitionStatus, error) {
	comp, err := uc.Get(ctx)
	if err != nil {
		return "", fmt.Errorf("CompetitionUseCase - GetStatus - Get: %w", err)
	}

	return comp.GetStatus(), nil
}

func (uc *CompetitionUseCase) IsSubmissionAllowed(ctx context.Context) (bool, error) {
	comp, err := uc.Get(ctx)
	if err != nil {
		return false, fmt.Errorf("CompetitionUseCase - IsSubmissionAllowed - Get: %w", err)
	}

	return comp.IsSubmissionAllowed(), nil
}
