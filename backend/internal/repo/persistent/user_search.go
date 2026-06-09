package persistent

import (
	"context"
	"fmt"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent/sqlc"
)

func (r *UserRepo) Search(ctx context.Context, search *string, limit, offset int) ([]*domain.User, error) {
	limit32, offset32, err := toLimitOffset(limit, offset)
	if err != nil {
		return nil, fmt.Errorf("UserRepo - Search: %w", err)
	}

	escapedSearch := EscapeSearchPtr(search)

	rows, err := r.Q(ctx).SearchUsers(ctx, sqlc.SearchUsersParams{
		Limit:  limit32,
		Offset: offset32,
		Search: escapedSearch,
	})
	if err != nil {
		return nil, fmt.Errorf("UserRepo - Search - scan: %w", err)
	}

	out := make([]*domain.User, 0, len(rows))
	for _, u := range rows {
		out = append(out, toDomainUser(userRowFrom(u)))
	}

	return out, nil
}

func (r *UserRepo) CountSearch(ctx context.Context, search *string) (int64, error) {
	escapedSearch := EscapeSearchPtr(search)

	count, err := r.Q(ctx).CountSearchUsers(ctx, escapedSearch)
	if err != nil {
		return 0, fmt.Errorf("UserRepo - CountSearch: %w", err)
	}

	return count, nil
}

func (r *UserRepo) SearchAdmin(ctx context.Context, filter repo.UserAdminSearchFilter, limit, offset int) ([]*domain.User, error) {
	limit32, offset32, err := toLimitOffset(limit, offset)
	if err != nil {
		return nil, fmt.Errorf("UserRepo - SearchAdmin: %w", err)
	}

	escapedSearch := EscapeSearchPtr(filter.Search)

	rows, err := r.Q(ctx).SearchUsersAdmin(ctx, sqlc.SearchUsersAdminParams{
		Limit:     limit32,
		Offset:    offset32,
		Search:    escapedSearch,
		BanStatus: string(filter.BanStatus),
	})
	if err != nil {
		return nil, fmt.Errorf("UserRepo - SearchAdmin - scan: %w", err)
	}

	out := make([]*domain.User, 0, len(rows))
	for _, u := range rows {
		out = append(out, toDomainUser(userRowFrom(u)))
	}

	return out, nil
}

func (r *UserRepo) CountSearchAdmin(ctx context.Context, filter repo.UserAdminSearchFilter) (int64, error) {
	escapedSearch := EscapeSearchPtr(filter.Search)

	count, err := r.Q(ctx).CountSearchUsersAdmin(ctx, sqlc.CountSearchUsersAdminParams{
		Search:    escapedSearch,
		BanStatus: string(filter.BanStatus),
	})
	if err != nil {
		return 0, fmt.Errorf("UserRepo - CountSearchAdmin: %w", err)
	}

	return count, nil
}

func (r *UserRepo) SearchByIP(ctx context.Context, ip string, limit, offset int) ([]*domain.User, error) {
	limit32, offset32, err := toLimitOffset(limit, offset)
	if err != nil {
		return nil, fmt.Errorf("UserRepo - SearchByIP: %w", err)
	}

	rows, err := r.Q(ctx).SearchUsersByIP(ctx, sqlc.SearchUsersByIPParams{
		Limit:  limit32,
		Offset: offset32,
		IP:     ip,
	})
	if err != nil {
		return nil, fmt.Errorf("UserRepo - SearchByIP - scan: %w", err)
	}

	out := make([]*domain.User, 0, len(rows))
	for _, u := range rows {
		out = append(out, toDomainUser(userRowFrom(u)))
	}

	return out, nil
}

func (r *UserRepo) CountSearchByIP(ctx context.Context, ip string) (int64, error) {
	count, err := r.Q(ctx).CountSearchUsersByIP(ctx, ip)
	if err != nil {
		return 0, fmt.Errorf("UserRepo - CountSearchByIP: %w", err)
	}

	return count, nil
}

func (r *UserRepo) SearchAdminByIP(ctx context.Context, ip string, banStatus repo.UserAdminBanStatus, limit, offset int) ([]*domain.User, error) {
	limit32, offset32, err := toLimitOffset(limit, offset)
	if err != nil {
		return nil, fmt.Errorf("UserRepo - SearchAdminByIP: %w", err)
	}

	rows, err := r.Q(ctx).SearchUsersAdminByIP(ctx, sqlc.SearchUsersAdminByIPParams{
		IP:        ip,
		Limit:     limit32,
		Offset:    offset32,
		BanStatus: string(banStatus),
	})
	if err != nil {
		return nil, fmt.Errorf("UserRepo - SearchAdminByIP - scan: %w", err)
	}

	out := make([]*domain.User, 0, len(rows))
	for _, u := range rows {
		out = append(out, toDomainUser(userRowFrom(u)))
	}

	return out, nil
}

func (r *UserRepo) CountSearchAdminByIP(ctx context.Context, ip string, banStatus repo.UserAdminBanStatus) (int64, error) {
	count, err := r.Q(ctx).CountSearchUsersAdminByIP(ctx, sqlc.CountSearchUsersAdminByIPParams{
		IP:        ip,
		BanStatus: string(banStatus),
	})
	if err != nil {
		return 0, fmt.Errorf("UserRepo - CountSearchAdminByIP: %w", err)
	}

	return count, nil
}

func (r *UserRepo) CountActiveUsers(ctx context.Context) (int64, error) {
	count, err := r.Q(ctx).CountActiveUsers(ctx)
	if err != nil {
		return 0, fmt.Errorf("UserRepo - CountActiveUsers: %w", err)
	}

	return count, nil
}
