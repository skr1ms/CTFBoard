package backup

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strconv"
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

var allowedCSVTables = map[string]bool{
	"users":       true,
	"teams":       true,
	"challenges":  true,
	"submissions": true,
	"solves":      true,
	"awards":      true,
}

func isAllowedCSVTable(table string) bool {
	return allowedCSVTables[table]
}

func csvExportUsers(users []*domain.User) ([]byte, error) {
	header := []string{"id", "username", "email", "password_hash", "role", "team_id", "is_verified", "verified_at", "is_banned", "banned_at", "banned_reason", "created_at"}

	rows := make([][]string, 0, len(users))
	for _, u := range users {
		teamID := ""

		if u.TeamID != nil {
			teamID = u.TeamID.String()
		}

		verifiedAt := ""

		if u.VerifiedAt != nil {
			verifiedAt = u.VerifiedAt.Format(time.RFC3339)
		}

		bannedAt := ""

		if u.BannedAt != nil {
			bannedAt = u.BannedAt.Format(time.RFC3339)
		}

		bannedReason := ""

		if u.BannedReason != nil {
			bannedReason = *u.BannedReason
		}

		rows = append(rows, []string{
			u.ID.String(),
			u.Username,
			u.Email,
			"",
			string(u.Role),
			teamID,
			strconv.FormatBool(u.IsVerified),
			verifiedAt,
			strconv.FormatBool(u.IsBanned),
			bannedAt,
			bannedReason,
			u.CreatedAt.Format(time.RFC3339),
		})
	}

	return writeCSV(header, rows)
}

func csvExportTeams(teams []*domain.Team) ([]byte, error) {
	header := []string{"id", "name", "captain_id", "invite_token", "invite_token_expires_at", "bracket_id", "is_solo", "is_banned", "banned_at", "banned_reason", "is_hidden", "created_at"}

	rows := make([][]string, 0, len(teams))
	for _, t := range teams {
		bracketID := ""

		if t.BracketID != nil {
			bracketID = t.BracketID.String()
		}

		inviteExpires := ""

		if t.InviteTokenExpiresAt != nil {
			inviteExpires = t.InviteTokenExpiresAt.Format(time.RFC3339)
		}

		bannedAt := ""

		if t.BannedAt != nil {
			bannedAt = t.BannedAt.Format(time.RFC3339)
		}

		bannedReason := ""

		if t.BannedReason != nil {
			bannedReason = *t.BannedReason
		}

		rows = append(rows, []string{
			t.ID.String(),
			t.Name,
			t.CaptainID.String(),
			t.InviteToken.String(),
			inviteExpires,
			bracketID,
			strconv.FormatBool(t.IsSolo),
			strconv.FormatBool(t.IsBanned),
			bannedAt,
			bannedReason,
			strconv.FormatBool(t.IsHidden),
			t.CreatedAt.Format(time.RFC3339),
		})
	}

	return writeCSV(header, rows)
}

func csvExportChallenges(challenges []*domain.Challenge) ([]byte, error) {
	header := []string{"id", "title", "description", "category", "flag_hash", "points", "initial_value", "min_value", "decay", "solve_count", "state", "connection_info", "max_attempts", "position", "is_regex", "is_case_insensitive", "flag_regex", "flag_format_regex"}

	rows := make([][]string, 0, len(challenges))
	for _, c := range challenges {
		flagFormat := ""

		if c.FlagFormatRegex != nil {
			flagFormat = *c.FlagFormatRegex
		}

		flagRegex := ""

		if c.FlagRegex != nil {
			flagRegex = *c.FlagRegex
		}

		rows = append(rows, []string{
			c.ID.String(),
			c.Title,
			c.Description,
			c.Category,
			c.FlagHash,
			strconv.Itoa(c.Points),
			strconv.Itoa(c.InitialValue),
			strconv.Itoa(c.MinValue),
			strconv.Itoa(c.Decay),
			strconv.Itoa(c.SolveCount),
			c.State,
			c.ConnectionInfo,
			strconv.Itoa(c.MaxAttempts),
			strconv.Itoa(c.Position),
			strconv.FormatBool(c.IsRegex),
			strconv.FormatBool(c.IsCaseInsensitive),
			flagRegex,
			flagFormat,
		})
	}

	return writeCSV(header, rows)
}

func csvExportSubmissions(subs []*domain.SubmissionWithDetails) ([]byte, error) {
	header := []string{"id", "user_id", "team_id", "challenge_id", "submitted_flag", "is_correct", "ip", "created_at", "submission_type", "banned_team_id", "banned_user_id"}

	rows := make([][]string, 0, len(subs))
	for _, s := range subs {
		teamID := ""

		if s.TeamID != nil {
			teamID = s.TeamID.String()
		}

		bannedTeamID := ""

		if s.BannedTeamID != nil {
			bannedTeamID = s.BannedTeamID.String()
		}

		bannedUserID := ""

		if s.BannedUserID != nil {
			bannedUserID = s.BannedUserID.String()
		}

		subType := s.Type
		if subType == "" {
			subType = domain.SubmissionTypeFromCorrect(s.IsCorrect)
		}

		rows = append(rows, []string{
			s.ID.String(),
			s.UserID.String(),
			teamID,
			s.ChallengeID.String(),
			s.SubmittedFlag,
			strconv.FormatBool(s.IsCorrect),
			s.IP,
			s.CreatedAt.Format(time.RFC3339),
			subType,
			bannedTeamID,
			bannedUserID,
		})
	}

	return writeCSV(header, rows)
}

func csvExportSolves(solves []*domain.Solve) ([]byte, error) {
	header := []string{"id", "user_id", "team_id", "challenge_id", "solved_at", "points_at_solve", "banned_team_id", "banned_user_id"}

	rows := make([][]string, 0, len(solves))
	for _, s := range solves {
		bannedTeamID := ""

		if s.BannedTeamID != nil {
			bannedTeamID = s.BannedTeamID.String()
		}

		bannedUserID := ""

		if s.BannedUserID != nil {
			bannedUserID = s.BannedUserID.String()
		}

		rows = append(rows, []string{
			s.ID.String(),
			s.UserID.String(),
			s.TeamID.String(),
			s.ChallengeID.String(),
			s.SolvedAt.Format(time.RFC3339),
			strconv.Itoa(s.PointsAtSolve),
			bannedTeamID,
			bannedUserID,
		})
	}

	return writeCSV(header, rows)
}

func csvExportAwards(awards []*domain.Award) ([]byte, error) {
	header := []string{"id", "team_id", "value", "description", "created_by", "created_at", "banned_team_id"}

	rows := make([][]string, 0, len(awards))
	for _, a := range awards {
		createdBy := ""

		if a.CreatedBy != nil {
			createdBy = a.CreatedBy.String()
		}

		bannedTeamID := ""

		if a.BannedTeamID != nil {
			bannedTeamID = a.BannedTeamID.String()
		}

		rows = append(rows, []string{
			a.ID.String(),
			a.TeamID.String(),
			strconv.Itoa(a.Value),
			a.Description,
			createdBy,
			a.CreatedAt.Format(time.RFC3339),
			bannedTeamID,
		})
	}

	return writeCSV(header, rows)
}

func writeCSV(header []string, rows [][]string) ([]byte, error) {
	var buf bytes.Buffer

	w := csv.NewWriter(&buf)

	err := w.Write(header)
	if err != nil {
		return nil, fmt.Errorf("BackupUseCase - writeCSV - write header: %w", err)
	}

	for _, row := range rows {
		err := w.Write(row)
		if err != nil {
			return nil, fmt.Errorf("BackupUseCase - writeCSV - write row: %w", err)
		}
	}

	w.Flush()

	err = w.Error()
	if err != nil {
		return nil, fmt.Errorf("BackupUseCase - writeCSV - flush csv: %w", err)
	}

	return buf.Bytes(), nil
}

func csvNormalizeUserRoles(header []string, rows [][]string) [][]string {
	roleIdx := -1

	for i, h := range header {
		if h == "role" {
			roleIdx = i

			break
		}
	}

	if roleIdx < 0 {
		return rows
	}

	out := make([][]string, len(rows))
	for i, row := range rows {
		rowCopy := make([]string, len(row))
		copy(rowCopy, row)

		if roleIdx < len(rowCopy) {
			rowCopy[roleIdx] = string(domain.RoleUser)
		}

		out[i] = rowCopy
	}

	return out
}

func parseCSV(data []byte) ([]string, [][]string, error) {
	r := csv.NewReader(bytes.NewReader(data))

	records, err := r.ReadAll()
	if err != nil {
		return nil, nil, fmt.Errorf("BackupUseCase - parseCSV - read csv: %w", err)
	}

	if len(records) < 1 {
		return nil, nil, httperr.ErrBackupCSVEmpty
	}

	return records[0], records[1:], nil
}
