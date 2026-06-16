package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/ashcloud/url-shortener/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type URLRepo struct {
	pool *pgxpool.Pool
}

func NewURLRepo(pool *pgxpool.Pool) *URLRepo {
	return &URLRepo{pool: pool}
}

func (r *URLRepo) Save(ctx context.Context, u *domain.URL) error {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO urls (short_code, original_url, user_id, expires_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at`,
		u.ShortCode, u.OriginalURL, u.UserID, u.ExpiresAt,
	)

	if err := row.Scan(&u.ID, &u.CreatedAt); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrCodeCollision
		}
		return fmt.Errorf("save url: %w", err)
	}

	return nil
}

func (r *URLRepo) GetByShortCode(ctx context.Context, code string) (*domain.URL, error) {
	var u domain.URL

	err := r.pool.QueryRow(ctx, `
		SELECT id, short_code, original_url, user_id, expires_at, click_count, created_at
		FROM urls
		WHERE short_code = $1`,
		code,
	).Scan(
		&u.ID, &u.ShortCode, &u.OriginalURL,
		&u.UserID, &u.ExpiresAt, &u.ClickCount, &u.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrURLNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get url by code %s: %w", code, err)
	}

	return &u, nil
}

func (r *URLRepo) IncrClickCount(ctx context.Context, code string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE urls SET click_count = click_count + 1 WHERE short_code = $1`,
		code,
	)
	if err != nil {
		return fmt.Errorf("incr click count: %w", err)
	}
	return nil
}

func (r *URLRepo) ListByUserID(ctx context.Context, userID int64) ([]*domain.URL, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, short_code, original_url, user_id, expires_at, click_count, created_at
		FROM urls
		WHERE user_id = $1
		ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list by user: %w", err)
	}
	defer rows.Close()

	var result []*domain.URL
	for rows.Next() {
		var u domain.URL
		if err = rows.Scan(
			&u.ID, &u.ShortCode, &u.OriginalURL,
			&u.UserID, &u.ExpiresAt, &u.ClickCount, &u.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan url row: %w", err)
		}
		result = append(result, &u)
	}

	return result, rows.Err()
}

func (r *URLRepo) Delete(ctx context.Context, code string, userID int64) error {
	res, err := r.pool.Exec(ctx,
		`DELETE FROM urls WHERE short_code = $1 AND user_id = $2`,
		code, userID,
	)
	if err != nil {
		return fmt.Errorf("delete url: %w", err)
	}
	if res.RowsAffected() == 0 {
		return domain.ErrURLNotFound
	}
	return nil
}
