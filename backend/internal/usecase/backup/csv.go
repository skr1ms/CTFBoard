package backup

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strconv"
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
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

func csvExportUsers(users []*entity.User) ([]byte, error) {
	header := []string{"id", "username", "email", "role", "is_verified", "team_id", "created_at"}
	rows := make([][]string, 0, len(users))
	for _, u := range users {
		teamID := ""
		if u.TeamID != nil {
			teamID = u.TeamID.String()
		}
		rows = append(rows, []string{
			u.ID.String(),
			u.Username,
			u.Email,
			u.Role,
			strconv.FormatBool(u.IsVerified),
			teamID,
			u.CreatedAt.Format(time.RFC3339),
		})
	}
	return writeCSV(header, rows)
}

func csvExportTeams(teams []*entity.Team) ([]byte, error) {
	header := []string{"id", "name", "captain_id", "invite_token", "bracket_id", "is_solo", "is_banned", "banned_at", "banned_reason", "is_hidden", "created_at"}
	rows := make([][]string, 0, len(teams))
	for _, t := range teams {
		bracketID := ""
		if t.BracketID != nil {
			bracketID = t.BracketID.String()
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

func csvExportChallenges(challenges []*entity.Challenge) ([]byte, error) {
	header := []string{"id", "title", "description", "category", "points", "initial_value", "min_value", "decay", "solve_count", "is_hidden"}
	rows := make([][]string, 0, len(challenges))
	for _, c := range challenges {
		rows = append(rows, []string{
			c.ID.String(),
			c.Title,
			c.Description,
			c.Category,
			strconv.Itoa(c.Points),
			strconv.Itoa(c.InitialValue),
			strconv.Itoa(c.MinValue),
			strconv.Itoa(c.Decay),
			strconv.Itoa(c.SolveCount),
			strconv.FormatBool(c.IsHidden),
		})
	}
	return writeCSV(header, rows)
}

func csvExportSubmissions(subs []*entity.SubmissionWithDetails) ([]byte, error) {
	header := []string{"id", "user_id", "team_id", "challenge_id", "submitted_flag", "is_correct", "ip", "created_at", "username", "team_name", "challenge_title"}
	rows := make([][]string, 0, len(subs))
	for _, s := range subs {
		teamID := ""
		if s.TeamID != nil {
			teamID = s.TeamID.String()
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
			s.Username,
			s.TeamName,
			s.ChallengeTitle,
		})
	}
	return writeCSV(header, rows)
}

func csvExportSolves(solves []*entity.Solve) ([]byte, error) {
	header := []string{"id", "user_id", "team_id", "challenge_id", "solved_at"}
	rows := make([][]string, 0, len(solves))
	for _, s := range solves {
		rows = append(rows, []string{
			s.ID.String(),
			s.UserID.String(),
			s.TeamID.String(),
			s.ChallengeID.String(),
			s.SolvedAt.Format(time.RFC3339),
		})
	}
	return writeCSV(header, rows)
}

func csvExportAwards(awards []*entity.Award) ([]byte, error) {
	header := []string{"id", "team_id", "value", "description", "created_by", "created_at"}
	rows := make([][]string, 0, len(awards))
	for _, a := range awards {
		createdBy := ""
		if a.CreatedBy != nil {
			createdBy = a.CreatedBy.String()
		}
		rows = append(rows, []string{
			a.ID.String(),
			a.TeamID.String(),
			strconv.Itoa(a.Value),
			a.Description,
			createdBy,
			a.CreatedAt.Format(time.RFC3339),
		})
	}
	return writeCSV(header, rows)
}

func writeCSV(header []string, rows [][]string) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	if err := w.Write(header); err != nil {
		return nil, fmt.Errorf("BackupUseCase - writeCSV - write header: %w", err)
	}
	for _, row := range rows {
		if err := w.Write(row); err != nil {
			return nil, fmt.Errorf("BackupUseCase - writeCSV - write row: %w", err)
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, fmt.Errorf("BackupUseCase - writeCSV - flush csv: %w", err)
	}
	return buf.Bytes(), nil
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
