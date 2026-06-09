package persistent

import (
	"context"
	"fmt"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent/sqlc"
)

func (r *TeamRepo) Search(ctx context.Context, search *string, limit, offset int) ([]*domain.Team, error) {
	limit32, offset32, err := toLimitOffset(limit, offset)
	if err != nil {
		return nil, fmt.Errorf("TeamRepo - Search: %w", err)
	}

	escapedSearch := EscapeSearchPtr(search)

	rows, err := r.Q(ctx).SearchTeams(ctx, sqlc.SearchTeamsParams{
		Limit:  limit32,
		Offset: offset32,
		Search: escapedSearch,
	})
	if err != nil {
		return nil, fmt.Errorf("TeamRepo - Search - scan: %w", err)
	}

	out := make([]*domain.Team, 0, len(rows))
	for _, row := range rows {
		out = append(out, toDomainTeam(teamRowFromSQLC(
			row.ID, row.Name, row.InviteToken,
			row.InviteTokenExpiresAt, row.BannedAt, row.CreatedAt,
			row.CaptainID, row.BracketID,
			row.IsSolo, row.IsAutoCreated, row.IsBanned, row.IsHidden,
			row.BannedReason, row.AvatarUrl,
		)))
	}

	return out, nil
}

func (r *TeamRepo) CountSearch(ctx context.Context, search *string) (int64, error) {
	escapedSearch := EscapeSearchPtr(search)

	count, err := r.Q(ctx).CountSearchTeams(ctx, escapedSearch)
	if err != nil {
		return 0, fmt.Errorf("TeamRepo - CountSearch: %w", err)
	}

	return count, nil
}

func (r *TeamRepo) SearchAdmin(ctx context.Context, filter repo.TeamAdminSearchFilter, limit, offset int) ([]*domain.Team, error) {
	limit32, offset32, err := toLimitOffset(limit, offset)
	if err != nil {
		return nil, fmt.Errorf("TeamRepo - SearchAdmin: %w", err)
	}

	escapedSearch := EscapeSearchPtr(filter.Search)

	rows, err := r.Q(ctx).SearchTeamsAdmin(ctx, sqlc.SearchTeamsAdminParams{
		Limit:      limit32,
		Offset:     offset32,
		Search:     escapedSearch,
		BanStatus:  string(filter.BanStatus),
		Visibility: string(filter.Visibility),
	})
	if err != nil {
		return nil, fmt.Errorf("TeamRepo - SearchAdmin - scan: %w", err)
	}

	out := make([]*domain.Team, 0, len(rows))
	for _, row := range rows {
		out = append(out, toDomainTeam(teamRowFromSQLC(
			row.ID, row.Name, row.InviteToken,
			row.InviteTokenExpiresAt, row.BannedAt, row.CreatedAt,
			row.CaptainID, row.BracketID,
			row.IsSolo, row.IsAutoCreated, row.IsBanned, row.IsHidden,
			row.BannedReason, row.AvatarUrl,
		)))
	}

	return out, nil
}

func (r *TeamRepo) CountSearchAdmin(ctx context.Context, filter repo.TeamAdminSearchFilter) (int64, error) {
	escapedSearch := EscapeSearchPtr(filter.Search)

	count, err := r.Q(ctx).CountSearchTeamsAdmin(ctx, sqlc.CountSearchTeamsAdminParams{
		Search:     escapedSearch,
		BanStatus:  string(filter.BanStatus),
		Visibility: string(filter.Visibility),
	})
	if err != nil {
		return 0, fmt.Errorf("TeamRepo - CountSearchAdmin: %w", err)
	}

	return count, nil
}
