package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/ashcloud/url-shortener/internal/domain"
	"github.com/ashcloud/url-shortener/internal/kafka"
	"github.com/ashcloud/url-shortener/pkg/hasher"
)

// URLRepository — интерфейс для работы с персистентным хранилищем URL.
type URLRepository interface {
	Save(ctx context.Context, u *domain.URL) error
	GetByShortCode(ctx context.Context, code string) (*domain.URL, error)
	IncrClickCount(ctx context.Context, code string) error
	ListByUserID(ctx context.Context, userID int64) ([]*domain.URL, error)
	Delete(ctx context.Context, code string, userID int64) error
}

// URLCache — интерфейс для кэша коротких ссылок в Redis.
type URLCache interface {
	Set(ctx context.Context, shortCode, originalURL string) error
	Get(ctx context.Context, shortCode string) (string, error)
	Delete(ctx context.Context, shortCode string) error
}

// ClickPublisher — интерфейс для публикации событий кликов.
type ClickPublisher interface {
	PublishClick(ctx context.Context, event kafka.ClickEvent) error
}

type URLService struct {
	repo           URLRepository
	cache          URLCache
	clickPublisher ClickPublisher
	log            *slog.Logger
	codeLen        int
	maxRetries     int
}

func NewURLService(
	repo URLRepository,
	cache URLCache,
	clickPublisher ClickPublisher,
	log *slog.Logger,
	codeLen int,
) *URLService {
	return &URLService{
		repo:           repo,
		cache:          cache,
		clickPublisher: clickPublisher,
		log:            log,
		codeLen:        codeLen,
		maxRetries:     5,
	}
}

// Create сокращает URL. При коллизии кода повторяет до maxRetries раз.
func (s *URLService) Create(ctx context.Context, req domain.CreateURLRequest) (*domain.URL, error) {
	if req.OriginalURL == "" {
		return nil, domain.ErrInvalidURL
	}

	code := req.CustomCode
	if code == "" {
		var err error
		code, err = hasher.Generate(s.codeLen)
		if err != nil {
			return nil, fmt.Errorf("generate code: %w", err)
		}
	}

	u := &domain.URL{
		ShortCode:   code,
		OriginalURL: req.OriginalURL,
		UserID:      req.UserID,
		ExpiresAt:   req.ExpiresAt,
	}

	for attempt := 0; attempt < s.maxRetries; attempt++ {
		err := s.repo.Save(ctx, u)
		if err == nil {
			break
		}

		if errors.Is(err, domain.ErrCodeCollision) && req.CustomCode == "" {
			s.log.Debug("short code collision, retrying", "attempt", attempt+1, "code", u.ShortCode)
			u.ShortCode, err = hasher.Generate(s.codeLen)
			if err != nil {
				return nil, fmt.Errorf("generate code on retry: %w", err)
			}
			continue
		}

		return nil, fmt.Errorf("save url: %w", err)
	}

	if err := s.cache.Set(ctx, u.ShortCode, u.OriginalURL); err != nil {
		s.log.Warn("failed to cache url after create", "code", u.ShortCode, "err", err)
	}

	return u, nil
}

// Resolve возвращает оригинальный URL и публикует click event в Kafka.
func (s *URLService) Resolve(ctx context.Context, shortCode string, r *http.Request) (string, error) {
	// Fast path — Redis кэш
	originalURL, err := s.cache.Get(ctx, shortCode)
	if err == nil {
		go s.publishClick(shortCode, r)
		go s.incrClickCount(shortCode)
		return originalURL, nil
	}

	if !errors.Is(err, domain.ErrURLNotFound) {
		s.log.Warn("redis get error, falling back to postgres", "code", shortCode, "err", err)
	}

	// Slow path — PostgreSQL
	u, err := s.repo.GetByShortCode(ctx, shortCode)
	if err != nil {
		return "", err
	}

	if u.IsExpired() {
		return "", domain.ErrURLExpired
	}

	if cacheErr := s.cache.Set(ctx, u.ShortCode, u.OriginalURL); cacheErr != nil {
		s.log.Warn("failed to cache url on resolve", "code", shortCode, "err", cacheErr)
	}

	go s.publishClick(shortCode, r)
	go s.incrClickCount(shortCode)

	return u.OriginalURL, nil
}

func (s *URLService) Delete(ctx context.Context, shortCode string, userID int64) error {
	if err := s.repo.Delete(ctx, shortCode, userID); err != nil {
		return err
	}
	if err := s.cache.Delete(ctx, shortCode); err != nil {
		s.log.Warn("failed to delete url from cache", "code", shortCode, "err", err)
	}
	return nil
}

func (s *URLService) ListByUser(ctx context.Context, userID int64) ([]*domain.URL, error) {
	return s.repo.ListByUserID(ctx, userID)
}

func (s *URLService) publishClick(shortCode string, r *http.Request) {
	event := kafka.ClickEvent{
		ShortCode: shortCode,
		IP:        r.RemoteAddr,
		UserAgent: r.UserAgent(),
		Referer:   r.Referer(),
		ClickedAt: time.Now().UTC(),
	}

	ctx := context.Background()
	if err := s.clickPublisher.PublishClick(ctx, event); err != nil {
		s.log.Warn("failed to publish click event", "code", shortCode, "err", err)
	}
}

func (s *URLService) incrClickCount(shortCode string) {
	ctx := context.Background()
	if err := s.repo.IncrClickCount(ctx, shortCode); err != nil {
		s.log.Warn("failed to increment click count", "code", shortCode, "err", err)
	}
}
