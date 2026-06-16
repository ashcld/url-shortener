package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/ashcloud/url-shortener/internal/domain"
	"github.com/go-chi/chi/v5"
)

type urlSvc interface {
	Create(ctx context.Context, req domain.CreateURLRequest) (*domain.URL, error)
	Resolve(ctx context.Context, shortCode string, r *http.Request) (string, error)
	Delete(ctx context.Context, shortCode string, userID int64) error
	ListByUser(ctx context.Context, userID int64) ([]*domain.URL, error)
}

type URLHandler struct {
	svc     urlSvc
	baseURL string
}

func NewURLHandler(svc urlSvc, baseURL string) *URLHandler {
	return &URLHandler{
		svc:     svc,
		baseURL: baseURL,
	}
}

type createRequest struct {
	URL        string `json:"url"`
	CustomCode string `json:"custom_code,omitempty"`
	ExpiresAt  string `json:"expires_at,omitempty"`
}

type createResponse struct {
	ShortCode   string `json:"short_code"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
	ExpiresAt   string `json:"expires_at,omitempty"`
	CreatedAt   string `json:"created_at"`
}

func (h *URLHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.URL == "" {
		writeError(w, http.StatusBadRequest, "url is required")
		return
	}

	createReq := domain.CreateURLRequest{
		OriginalURL: req.URL,
		CustomCode:  req.CustomCode,
	}

	if req.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, req.ExpiresAt)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid expires_at format, use RFC3339")
			return
		}
		createReq.ExpiresAt = &t
	}

	url, err := h.svc.Create(r.Context(), createReq)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidURL) {
			writeError(w, http.StatusBadRequest, "invalid url")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create url")
		return
	}

	resp := createResponse{
		ShortCode:   url.ShortCode,
		ShortURL:    h.baseURL + "/" + url.ShortCode,
		OriginalURL: url.OriginalURL,
		CreatedAt:   url.CreatedAt.Format(time.RFC3339),
	}
	if url.ExpiresAt != nil {
		resp.ExpiresAt = url.ExpiresAt.Format(time.RFC3339)
	}

	writeJSON(w, http.StatusCreated, resp)
}

func (h *URLHandler) Redirect(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")

	originalURL, err := h.svc.Resolve(r.Context(), code, r)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrURLNotFound):
			writeError(w, http.StatusNotFound, "url not found")
		case errors.Is(err, domain.ErrURLExpired):
			writeError(w, http.StatusGone, "url expired")
		default:
			writeError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	http.Redirect(w, r, originalURL, http.StatusMovedPermanently)
}

func (h *URLHandler) Delete(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")

	userIDStr := r.URL.Query().Get("user_id")
	if userIDStr == "" {
		writeError(w, http.StatusBadRequest, "user_id is required")
		return
	}
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user_id")
		return
	}

	if err = h.svc.Delete(r.Context(), code, userID); err != nil {
		if errors.Is(err, domain.ErrURLNotFound) {
			writeError(w, http.StatusNotFound, "url not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to delete url")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type urlResponse struct {
	ShortCode   string `json:"short_code"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
	ClickCount  int64  `json:"click_count"`
	CreatedAt   string `json:"created_at"`
	ExpiresAt   string `json:"expires_at,omitempty"`
}

func (h *URLHandler) List(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get("user_id")
	if userIDStr == "" {
		writeError(w, http.StatusBadRequest, "user_id is required")
		return
	}
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user_id")
		return
	}

	urls, err := h.svc.ListByUser(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list urls")
		return
	}

	resp := make([]urlResponse, 0, len(urls))
	for _, u := range urls {
		item := urlResponse{
			ShortCode:   u.ShortCode,
			ShortURL:    h.baseURL + "/" + u.ShortCode,
			OriginalURL: u.OriginalURL,
			ClickCount:  u.ClickCount,
			CreatedAt:   u.CreatedAt.Format(time.RFC3339),
		}
		if u.ExpiresAt != nil {
			item.ExpiresAt = u.ExpiresAt.Format(time.RFC3339)
		}
		resp = append(resp, item)
	}

	writeJSON(w, http.StatusOK, resp)
}

type errorResponse struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorResponse{Error: msg})
}
