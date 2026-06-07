package competition

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/guard"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/crypto"
)

const (
	shareConfigEnabled        = "social_shares_enabled"
	shareConfigCTFName        = "ctf_name"
	shareConfigCTFDescription = "ctf_description"
	shareConfigCTFLogo        = "ctf_logo"
)

type ShareUseCase struct {
	deps ShareDeps
}

type ShareDeps struct {
	SolveRepo     repo.SolveRepository
	ChallengeRepo repo.ChallengeRepository
	UserRepo      repo.UserRepository
	TeamRepo      repo.TeamRepository
	CompParamUC   shareConfigGetter
	BaseURL       string
	FrontendURL   string
	ShareSecret   string
}

type shareConfigGetter interface {
	GetString(ctx context.Context, key, defaultVal string) string
	GetBool(ctx context.Context, key string, defaultVal bool) bool
}

var _ usecase.ShareUseCase = (*ShareUseCase)(nil)

func NewShareUseCase(deps ShareDeps) *ShareUseCase {
	return &ShareUseCase{deps: deps}
}

func (uc *ShareUseCase) CreateSolveShare(ctx context.Context, params usecase.CreateShareParams) (*usecase.ShareLink, error) {
	if params.Type != usecase.ShareTypeSolve {
		return nil, apperr.NewValidationErrorf("share type must be solve")
	}

	if !uc.sharesEnabled(ctx) {
		return nil, apperr.ErrSharesDisabled
	}

	if params.TeamID == uuid.Nil {
		return nil, apperr.ErrUserNotInTeam
	}

	challenge, err := uc.deps.ChallengeRepo.GetByID(ctx, params.ChallengeID)
	if err != nil {
		return nil, fmt.Errorf("ShareUseCase - CreateSolveShare - ChallengeRepo.GetByID: %w", err)
	}

	if err := guard.EnsureChallengeVisible(challenge); err != nil {
		return nil, err
	}

	solve, err := uc.deps.SolveRepo.GetByTeamAndChallenge(ctx, params.TeamID, params.ChallengeID)
	if err != nil {
		if errors.Is(err, apperr.ErrSolveNotFound) {
			return nil, apperr.ErrSolutionAccessDenied
		}

		return nil, fmt.Errorf("ShareUseCase - CreateSolveShare - SolveRepo.GetByTeamAndChallenge: %w", err)
	}

	return &usecase.ShareLink{
		Type:    usecase.ShareTypeSolve,
		URL:     uc.solveShareURL(solve.ID),
		SolveID: solve.ID,
	}, nil
}

func (uc *ShareUseCase) ResolveSolveShare(ctx context.Context, solveID uuid.UUID, mac string) (*usecase.SolveShare, error) {
	if !uc.sharesEnabled(ctx) {
		return nil, apperr.ErrSharesDisabled
	}

	if !uc.verifySolveMAC(solveID, mac) {
		return nil, apperr.ErrShareNotFound
	}

	solve, err := uc.deps.SolveRepo.GetByID(ctx, solveID)
	if err != nil {
		if errors.Is(err, apperr.ErrSolveNotFound) {
			return nil, apperr.ErrShareNotFound
		}

		return nil, fmt.Errorf("ShareUseCase - ResolveSolveShare - SolveRepo.GetByID: %w", err)
	}

	if solve.BannedTeamID != nil || solve.BannedUserID != nil {
		return nil, apperr.ErrShareNotFound
	}

	challenge, err := uc.deps.ChallengeRepo.GetByID(ctx, solve.ChallengeID)
	if err != nil {
		if errors.Is(err, apperr.ErrChallengeNotFound) || errors.Is(err, apperr.ErrVisibilityForbidden) {
			return nil, apperr.ErrShareNotFound
		}

		return nil, fmt.Errorf("ShareUseCase - ResolveSolveShare - ChallengeRepo.GetByID: %w", err)
	}

	if challenge.State == domain.ChallengeStateHidden {
		return nil, apperr.ErrShareNotFound
	}

	team, err := uc.deps.TeamRepo.GetByID(ctx, solve.TeamID)
	if err != nil {
		if errors.Is(err, apperr.ErrTeamNotFound) {
			return nil, apperr.ErrShareNotFound
		}

		return nil, fmt.Errorf("ShareUseCase - ResolveSolveShare - TeamRepo.GetByID: %w", err)
	}

	if team.IsBanned || team.IsHidden {
		return nil, apperr.ErrShareNotFound
	}

	user, err := uc.deps.UserRepo.GetByID(ctx, solve.UserID)
	if err != nil {
		if errors.Is(err, apperr.ErrUserNotFound) {
			return nil, apperr.ErrShareNotFound
		}

		return nil, fmt.Errorf("ShareUseCase - ResolveSolveShare - UserRepo.GetByID: %w", err)
	}

	if user.IsBanned {
		return nil, apperr.ErrShareNotFound
	}

	return &usecase.SolveShare{
		SolveID:           solve.ID,
		ChallengeID:       solve.ChallengeID,
		TeamID:            solve.TeamID,
		UserID:            solve.UserID,
		TeamName:          team.Name,
		Username:          user.Username,
		ChallengeTitle:    challenge.Title,
		ChallengeCategory: challenge.Category,
		CTFName:           uc.configString(ctx, shareConfigCTFName, "CTF Platform"),
		CTFDescription:    uc.configString(ctx, shareConfigCTFDescription, ""),
		CTFLogo:           uc.configString(ctx, shareConfigCTFLogo, ""),
		RegisterURL:       uc.registerURL(),
		PointsAtSolve:     solve.PointsAtSolve,
		SolvedAt:          solve.SolvedAt,
	}, nil
}

func (uc *ShareUseCase) sharesEnabled(ctx context.Context) bool {
	if uc.deps.CompParamUC == nil {
		return true
	}

	return uc.deps.CompParamUC.GetBool(ctx, shareConfigEnabled, true)
}

func (uc *ShareUseCase) configString(ctx context.Context, key, fallback string) string {
	if uc.deps.CompParamUC == nil {
		return fallback
	}

	return uc.deps.CompParamUC.GetString(ctx, key, fallback)
}

func (uc *ShareUseCase) solveShareURL(solveID uuid.UUID) string {
	baseURL := strings.TrimRight(uc.deps.BaseURL, "/")
	if baseURL == "" {
		baseURL = "/api/v1"
	}

	values := url.Values{}
	values.Set("solve_id", solveID.String())
	values.Set("mac", uc.signSolve(solveID))

	return baseURL + "/api/v1/shares/solve?" + values.Encode()
}

func (uc *ShareUseCase) registerURL() string {
	frontendURL := strings.TrimRight(uc.deps.FrontendURL, "/")
	if frontendURL == "" {
		return ""
	}

	return frontendURL + "/register"
}

func (uc *ShareUseCase) signSolve(solveID uuid.UUID) string {
	return hex.EncodeToString(crypto.HMACSign([]byte(uc.deps.ShareSecret), []byte(solveShareMessage(solveID))))
}

func (uc *ShareUseCase) verifySolveMAC(solveID uuid.UUID, macHex string) bool {
	mac, err := hex.DecodeString(macHex)
	if err != nil {
		return false
	}

	return crypto.HMACVerify([]byte(uc.deps.ShareSecret), []byte(solveShareMessage(solveID)), mac)
}

func solveShareMessage(solveID uuid.UUID) string {
	return usecase.ShareTypeSolve + ":" + solveID.String()
}
