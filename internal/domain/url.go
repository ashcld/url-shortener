package domain

import "time"

// URL — основная сущность. Хранит оригинальный URL и его короткий код.
type URL struct {
	ID          int64
	ShortCode   string
	OriginalURL string
	UserID      *int64
	ExpiresAt   *time.Time
	CreatedAt   time.Time
	ClickCount  int64
}

// IsExpired проверяет что срок действия ссылки истёк.
func (u *URL) IsExpired() bool {
	if u.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*u.ExpiresAt)
}

// Click — событие клика по короткой ссылке.
type Click struct {
	ShortCode string
	IP        string
	UserAgent string
	Referer   string
	Country   string
	Device    string
	ClickedAt time.Time
}

// CreateURLRequest — входные данные для создания короткой ссылки.
type CreateURLRequest struct {
	OriginalURL string
	UserID      *int64
	ExpiresAt   *time.Time
	CustomCode  string
}

// URLStats — агрегированная аналитика по короткой ссылке.
type URLStats struct {
	ShortCode   string
	TotalClicks int64
	UniqueIPs   int64
	ByDevice    map[string]int64
	ByCountry   map[string]int64
	ByDate      []DailyStats
}

type DailyStats struct {
	Date   string
	Clicks int64
}
