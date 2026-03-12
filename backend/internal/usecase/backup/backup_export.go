package backup

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/logger"
)

const exportTimeout = 5 * time.Minute

func (uc *BackupUseCase) Export(ctx context.Context, opts entity.ExportOptions) (*entity.BackupData, error) {
	ctx, cancel := context.WithTimeout(ctx, exportTimeout)
	defer cancel()

	g, gCtx := errgroup.WithContext(ctx)
	backup := &entity.BackupData{
		Version:    entity.BackupVersion,
		ExportedAt: time.Now().UTC(),
	}
	var mu sync.Mutex

	g.Go(func() error {
		comp, err := uc.deps.CompetitionRepo.Get(gCtx)
		if err != nil {
			return fmt.Errorf("BackupUseCase - Export - CompetitionRepo.Get: %w", err)
		}
		mu.Lock()
		backup.Competition = comp
		mu.Unlock()
		return nil
	})

	g.Go(func() error {
		challenges, err := uc.fetchChallengesWithHints(gCtx)
		if err != nil {
			return fmt.Errorf("BackupUseCase - Export - fetchChallengesWithHints: %w", err)
		}
		if uc.deps.TagRepo != nil {
			challengeIDs := make([]uuid.UUID, len(challenges))
			for i, c := range challenges {
				challengeIDs[i] = c.ID
			}
			tagsByChallenge, err := uc.deps.TagRepo.GetByChallengeIDs(gCtx, challengeIDs)
			if err != nil {
				return fmt.Errorf("BackupUseCase - Export - TagRepo.GetByChallengeIDs: %w", err)
			}
			seenTags := make(map[uuid.UUID]entity.Tag)
			for i := range challenges {
				tags := tagsByChallenge[challenges[i].ID]
				tagIDs := make([]uuid.UUID, 0, len(tags))
				for _, t := range tags {
					seenTags[t.ID] = *t
					tagIDs = append(tagIDs, t.ID)
				}
				challenges[i].TagIDs = tagIDs
			}
			uniqueTags := make([]entity.Tag, 0, len(seenTags))
			for _, t := range seenTags {
				uniqueTags = append(uniqueTags, t)
			}
			mu.Lock()
			backup.Tags = uniqueTags
			mu.Unlock()
		}
		mu.Lock()
		backup.Challenges = challenges
		mu.Unlock()
		return nil
	})

	uc.exportBrackets(gCtx, backup, &mu, g)
	uc.exportChallengeRequirements(gCtx, backup, &mu, g)
	uc.exportSolutions(gCtx, backup, &mu, g)
	uc.exportComments(gCtx, backup, &mu, g)
	uc.exportFields(gCtx, backup, &mu, g)
	uc.exportFieldValues(gCtx, backup, &mu, g)
	uc.exportOptional(gCtx, backup, opts, &mu, g)

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("BackupUseCase - Export - errgroup.Wait: %w", err)
	}
	return backup, nil
}

func (uc *BackupUseCase) exportBrackets(ctx context.Context, backup *entity.BackupData, mu *sync.Mutex, g *errgroup.Group) {
	if uc.deps.BracketRepo == nil {
		return
	}
	g.Go(func() error {
		list, err := uc.deps.BracketRepo.GetAll(ctx)
		if err != nil {
			return fmt.Errorf("BackupUseCase - Export - BracketRepo.GetAll: %w", err)
		}
		out := make([]entity.Bracket, len(list))
		for i, b := range list {
			out[i] = *b
		}
		mu.Lock()
		backup.Brackets = out
		mu.Unlock()
		return nil
	})
}

func (uc *BackupUseCase) exportChallengeRequirements(ctx context.Context, backup *entity.BackupData, mu *sync.Mutex, g *errgroup.Group) {
	g.Go(func() error {
		list, err := uc.deps.ChallengeRepo.GetAllRequirementPairs(ctx)
		if err != nil {
			return fmt.Errorf("BackupUseCase - Export - ChallengeRepo.GetAllRequirementPairs: %w", err)
		}
		out := make([]entity.ChallengeRequirementPair, len(list))
		for i, p := range list {
			out[i] = *p
		}
		mu.Lock()
		backup.ChallengeRequirements = out
		mu.Unlock()
		return nil
	})
}

func (uc *BackupUseCase) exportSolutions(ctx context.Context, backup *entity.BackupData, mu *sync.Mutex, g *errgroup.Group) {
	g.Go(func() error {
		list, err := uc.deps.ChallengeRepo.GetAllSolutions(ctx)
		if err != nil {
			return fmt.Errorf("BackupUseCase - Export - ChallengeRepo.GetAllSolutions: %w", err)
		}
		out := make([]entity.SolutionBackup, len(list))
		for i, s := range list {
			out[i] = *s
		}
		mu.Lock()
		backup.Solutions = out
		mu.Unlock()
		return nil
	})
}

func (uc *BackupUseCase) exportComments(ctx context.Context, backup *entity.BackupData, mu *sync.Mutex, g *errgroup.Group) {
	if uc.deps.CommentRepo == nil {
		return
	}
	g.Go(func() error {
		list, err := uc.deps.CommentRepo.GetAll(ctx)
		if err != nil {
			return fmt.Errorf("BackupUseCase - Export - CommentRepo.GetAll: %w", err)
		}
		out := make([]entity.Comment, len(list))
		for i, c := range list {
			out[i] = *c
		}
		mu.Lock()
		backup.Comments = out
		mu.Unlock()
		return nil
	})
}

func (uc *BackupUseCase) exportFields(ctx context.Context, backup *entity.BackupData, mu *sync.Mutex, g *errgroup.Group) {
	if uc.deps.FieldRepo == nil {
		return
	}
	g.Go(func() error {
		list, err := uc.deps.FieldRepo.GetAll(ctx)
		if err != nil {
			return fmt.Errorf("BackupUseCase - Export - FieldRepo.GetAll: %w", err)
		}
		out := make([]entity.Field, len(list))
		for i, f := range list {
			out[i] = *f
		}
		mu.Lock()
		backup.Fields = out
		mu.Unlock()
		return nil
	})
}

func (uc *BackupUseCase) exportFieldValues(ctx context.Context, backup *entity.BackupData, mu *sync.Mutex, g *errgroup.Group) {
	if uc.deps.FieldValueRepo == nil {
		return
	}
	g.Go(func() error {
		list, err := uc.deps.FieldValueRepo.GetAll(ctx)
		if err != nil {
			return fmt.Errorf("BackupUseCase - Export - FieldValueRepo.GetAll: %w", err)
		}
		out := make([]entity.FieldValue, len(list))
		for i, v := range list {
			out[i] = *v
		}
		mu.Lock()
		backup.FieldValues = out
		mu.Unlock()
		return nil
	})
}

func (uc *BackupUseCase) exportOptional(
	ctx context.Context,
	backup *entity.BackupData,
	opts entity.ExportOptions,
	mu *sync.Mutex,
	g *errgroup.Group,
) {
	uc.exportOptionalTeams(ctx, backup, opts, mu, g)
	uc.exportOptionalUsers(ctx, backup, opts, mu, g)
	uc.exportOptionalAwards(ctx, backup, opts, mu, g)
	uc.exportOptionalSolves(ctx, backup, opts, mu, g)
	uc.exportOptionalHintUnlocks(ctx, backup, opts, mu, g)
	uc.exportOptionalFiles(ctx, backup, opts, mu, g)
}

func (uc *BackupUseCase) exportOptionalTeams(ctx context.Context, backup *entity.BackupData, opts entity.ExportOptions, mu *sync.Mutex, g *errgroup.Group) {
	if !opts.IncludeTeams {
		return
	}
	g.Go(func() error {
		teams, err := uc.fetchTeamsWithMembers(ctx)
		if err != nil {
			return fmt.Errorf("BackupUseCase - Export - fetchTeamsWithMembers: %w", err)
		}
		mu.Lock()
		backup.Teams = teams
		mu.Unlock()
		return nil
	})
}

func (uc *BackupUseCase) exportOptionalUsers(ctx context.Context, backup *entity.BackupData, opts entity.ExportOptions, mu *sync.Mutex, g *errgroup.Group) {
	if !opts.IncludeUsers {
		return
	}
	g.Go(func() error {
		users, err := uc.fetchUsers(ctx)
		if err != nil {
			return fmt.Errorf("BackupUseCase - Export - fetchUsers: %w", err)
		}
		mu.Lock()
		backup.Users = users
		mu.Unlock()
		return nil
	})
}

func (uc *BackupUseCase) exportOptionalAwards(ctx context.Context, backup *entity.BackupData, opts entity.ExportOptions, mu *sync.Mutex, g *errgroup.Group) {
	if !opts.IncludeAwards {
		return
	}
	g.Go(func() error {
		awards, err := uc.fetchAwards(ctx)
		if err != nil {
			return fmt.Errorf("BackupUseCase - Export - fetchAwards: %w", err)
		}
		mu.Lock()
		backup.Awards = awards
		mu.Unlock()
		return nil
	})
}

func (uc *BackupUseCase) exportOptionalSolves(ctx context.Context, backup *entity.BackupData, opts entity.ExportOptions, mu *sync.Mutex, g *errgroup.Group) {
	if !opts.IncludeSolves {
		return
	}
	g.Go(func() error {
		solves, err := uc.fetchSolves(ctx)
		if err != nil {
			return fmt.Errorf("BackupUseCase - Export - fetchSolves: %w", err)
		}
		mu.Lock()
		backup.Solves = solves
		mu.Unlock()
		return nil
	})
}

func (uc *BackupUseCase) exportOptionalHintUnlocks(ctx context.Context, backup *entity.BackupData, opts entity.ExportOptions, mu *sync.Mutex, g *errgroup.Group) {
	if !opts.IncludeHintUnlocks {
		return
	}
	g.Go(func() error {
		unlocks, err := uc.deps.HintRepo.GetAllUnlocksForBackup(ctx)
		if err != nil {
			return fmt.Errorf("BackupUseCase - Export - HintRepo.GetAllUnlocksForBackup: %w", err)
		}
		result := make([]entity.HintUnlock, len(unlocks))
		for i, u := range unlocks {
			result[i] = *u
		}
		mu.Lock()
		backup.HintUnlocks = result
		mu.Unlock()
		return nil
	})
}

func (uc *BackupUseCase) exportOptionalFiles(ctx context.Context, backup *entity.BackupData, opts entity.ExportOptions, mu *sync.Mutex, g *errgroup.Group) {
	if !opts.IncludeFiles {
		return
	}
	g.Go(func() error {
		files, err := uc.fetchFiles(ctx)
		if err != nil {
			return fmt.Errorf("BackupUseCase - Export - fetchFiles: %w", err)
		}
		mu.Lock()
		backup.Files = files
		mu.Unlock()
		return nil
	})
}

func (uc *BackupUseCase) fetchChallengesWithHints(ctx context.Context) ([]entity.ChallengeExport, error) {
	challengesWithSolved, err := uc.deps.ChallengeRepo.GetAll(ctx, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("BackupUseCase - fetchChallengesWithHints - ChallengeRepo.GetAll: %w", err)
	}
	challengeIDs := make([]uuid.UUID, len(challengesWithSolved))
	for i, cws := range challengesWithSolved {
		challengeIDs[i] = cws.Challenge.ID
	}
	var hintsByChallenge map[uuid.UUID][]*entity.Hint
	if uc.deps.HintRepo != nil {
		var err error
		hintsByChallenge, err = uc.deps.HintRepo.GetByChallengeIDs(ctx, challengeIDs)
		if err != nil {
			return nil, fmt.Errorf("BackupUseCase - fetchChallengesWithHints - HintRepo.GetByChallengeIDs: %w", err)
		}
	}
	if hintsByChallenge == nil {
		hintsByChallenge = make(map[uuid.UUID][]*entity.Hint)
	}
	result := make([]entity.ChallengeExport, len(challengesWithSolved))
	for i, cws := range challengesWithSolved {
		hints := hintsByChallenge[cws.Challenge.ID]
		hintsCopy := make([]entity.Hint, len(hints))
		for j, h := range hints {
			hintsCopy[j] = *h
		}
		result[i] = entity.ChallengeExport{
			Challenge: *cws.Challenge,
			FlagHash:  cws.Challenge.FlagHash,
			FlagRegex: cws.Challenge.FlagRegex,
			Hints:     hintsCopy,
		}
	}
	return result, nil
}

func (uc *BackupUseCase) fetchTeamsWithMembers(ctx context.Context) ([]entity.TeamExport, error) {
	teams, err := uc.deps.TeamRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("BackupUseCase - fetchTeamsWithMembers - TeamRepo.GetAll: %w", err)
	}
	teamIDs := make([]uuid.UUID, len(teams))
	for i, t := range teams {
		teamIDs[i] = t.ID
	}
	membersByTeam, err := uc.deps.UserRepo.GetByTeamIDs(ctx, teamIDs)
	if err != nil {
		return nil, fmt.Errorf("BackupUseCase - fetchTeamsWithMembers - UserRepo.GetByTeamIDs: %w", err)
	}
	result := make([]entity.TeamExport, len(teams))
	for i, team := range teams {
		members := membersByTeam[team.ID]
		memberIDs := make([]uuid.UUID, len(members))
		for j, m := range members {
			memberIDs[j] = m.ID
		}
		result[i] = entity.TeamExport{
			Team:                 *team,
			InviteToken:          team.InviteToken,
			InviteTokenExpiresAt: team.InviteTokenExpiresAt,
			MemberIDs:            memberIDs,
		}
	}
	return result, nil
}

func (uc *BackupUseCase) fetchUsers(ctx context.Context) ([]entity.UserExport, error) {
	users, err := uc.deps.UserRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("BackupUseCase - fetchUsers - UserRepo.GetAll: %w", err)
	}

	result := make([]entity.UserExport, 0, len(users))
	for _, u := range users {
		result = append(result, entity.UserExport{
			ID:           u.ID,
			Username:     u.Username,
			Email:        u.Email,
			Role:         string(u.Role),
			TeamID:       u.TeamID,
			IsVerified:   u.IsVerified,
			VerifiedAt:   u.VerifiedAt,
			IsBanned:     u.IsBanned,
			BannedAt:     u.BannedAt,
			BannedReason: u.BannedReason,
			CreatedAt:    u.CreatedAt,
		})
	}
	return result, nil
}

func (uc *BackupUseCase) fetchAwards(ctx context.Context) ([]entity.Award, error) {
	awards, err := uc.deps.AwardRepo.GetAllForBackup(ctx)
	if err != nil {
		return nil, fmt.Errorf("BackupUseCase - fetchAwards - AwardRepo.GetAllForBackup: %w", err)
	}

	result := make([]entity.Award, len(awards))
	for i, a := range awards {
		result[i] = *a
	}

	return result, nil
}

func (uc *BackupUseCase) fetchSolves(ctx context.Context) ([]entity.Solve, error) {
	solves, err := uc.deps.SolveRepo.GetAllForBackup(ctx)
	if err != nil {
		return nil, fmt.Errorf("BackupUseCase - fetchSolves - SolveRepo.GetAllForBackup: %w", err)
	}

	result := make([]entity.Solve, len(solves))
	for i, s := range solves {
		result[i] = *s
	}

	return result, nil
}

func (uc *BackupUseCase) fetchFiles(ctx context.Context) ([]entity.File, error) {
	files, err := uc.deps.FileRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("BackupUseCase - fetchFiles - FileRepo.GetAll: %w", err)
	}

	result := make([]entity.File, len(files))
	for i, f := range files {
		result[i] = *f
	}

	return result, nil
}

func (uc *BackupUseCase) ExportZIP(ctx context.Context, opts entity.ExportOptions) (io.ReadCloser, error) {
	pr, pw := io.Pipe()
	go uc.exportZIPWorker(ctx, pw, opts)
	return pr, nil
}

func (uc *BackupUseCase) exportZIPWorker(ctx context.Context, pw *io.PipeWriter, opts entity.ExportOptions) {
	defer pw.Close()
	select {
	case <-ctx.Done():
		pw.CloseWithError(ctx.Err())
		return
	default:
	}
	zw := zip.NewWriter(pw)
	defer zw.Close()
	data, err := uc.Export(ctx, opts)
	if err != nil {
		pw.CloseWithError(err)
		return
	}
	if ctx.Err() != nil {
		pw.CloseWithError(ctx.Err())
		return
	}
	if err := uc.writeBackupJSON(zw, data); err != nil {
		pw.CloseWithError(err)
		return
	}
	if opts.IncludeFiles && len(data.Files) > 0 {
		if ctx.Err() != nil {
			pw.CloseWithError(ctx.Err())
			return
		}
		skipped := uc.streamFilesToZip(ctx, zw, data.Files)
		uc.deps.Logger.Info("BackupUseCase - ExportZIP - completed", logger.Fields{
			"challenges": len(data.Challenges),
			"teams":      len(data.Teams),
			"files":      len(data.Files),
			"skipped":    skipped,
		})
	} else {
		uc.deps.Logger.Info("BackupUseCase - ExportZIP - completed", logger.Fields{
			"challenges": len(data.Challenges),
			"teams":      len(data.Teams),
			"files":      0,
		})
	}
}

func (uc *BackupUseCase) writeBackupJSON(zw *zip.Writer, data *entity.BackupData) error {
	jsonFile, err := zw.Create("backup.json")
	if err != nil {
		return fmt.Errorf("BackupUseCase - ExportZIP - create backup.json: %w", err)
	}
	if err := json.NewEncoder(jsonFile).Encode(data); err != nil {
		return fmt.Errorf("BackupUseCase - ExportZIP - encode backup.json: %w", err)
	}
	readme, err := zw.Create("README.md")
	if err != nil {
		return fmt.Errorf("BackupUseCase - ExportZIP - create README.md: %w", err)
	}
	if _, err := fmt.Fprintf(readme, "# AstroCTFb Backup\n\nBackup created: %s\nVersion: %s\n", data.ExportedAt.Format(time.RFC3339), data.Version); err != nil {
		return fmt.Errorf("BackupUseCase - ExportZIP - write README: %w", err)
	}
	return nil
}

func (uc *BackupUseCase) streamFilesToZip(ctx context.Context, zw *zip.Writer, files []entity.File) int {
	var skipped int
	for _, file := range files {
		if ctx.Err() != nil {
			break
		}
		path := fmt.Sprintf("files/challenge-%s/%s", file.ChallengeID, filepath.Base(file.Filename))
		f, err := zw.Create(path)
		if err != nil {
			uc.deps.Logger.WithError(err).WithFields(logger.Fields{"file": file.Filename}).Warn("BackupUseCase - streamFilesToZip - create")
			skipped++
			continue
		}

		rc, err := uc.deps.Storage.Download(ctx, file.Location)
		if err != nil {
			uc.deps.Logger.WithError(err).WithFields(logger.Fields{"file": file.Filename, "location": file.Location}).Warn("BackupUseCase - streamFilesToZip - download")
			skipped++
			continue
		}
		func() {
			defer func() { _ = rc.Close() }()
			if _, err := io.Copy(f, rc); err != nil {
				uc.deps.Logger.WithError(err).WithFields(logger.Fields{"file": file.Filename}).Warn("BackupUseCase - streamFilesToZip - copy")
				skipped++
			}
		}()
	}

	if skipped > 0 {
		uc.deps.Logger.Warn("BackupUseCase - streamFilesToZip - completed with skipped files", logger.Fields{
			"total":   len(files),
			"skipped": skipped,
		})
	}

	return skipped
}
