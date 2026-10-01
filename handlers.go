package main

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Handler struct {
	store     LinkStorer
	users     UserStorer
	clicks    *ClickLogger
	analytics AnalyticsReader
	jwtSecret []byte
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token string `json:"token"`
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	if req.Email == "" || len(req.Password) < 8 {
		http.Error(w, "email is required and password must be at least 8 characters", http.StatusBadRequest)
		return
	}

	hash, err := HashPassword(req.Password)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if _, err := h.users.Create(r.Context(), req.Email, hash); err != nil {
		if errors.Is(err, ErrEmailTaken) {
			http.Error(w, "email already registered", http.StatusConflict)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	user, err := h.users.GetByEmail(r.Context(), req.Email)
	if err != nil {
		http.Error(w, "invalid email or password", http.StatusUnauthorized)
		return
	}

	if err := CheckPassword(user.PasswordHash, req.Password); err != nil {
		http.Error(w, "invalid email or password", http.StatusUnauthorized)
		return
	}

	token, err := GenerateJWT(user.ID, h.jwtSecret, 24*time.Hour)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(loginResponse{Token: token})
}

type createLinkRequest struct {
	URL string `json:"url"`
}

type createLinkResponse struct {
	ShortCode string `json:"short_code"`
	ShortURL  string `json:"short_url"`
}

func (h *Handler) CreateShortLink(w http.ResponseWriter, r *http.Request) {
	var req createLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	if !isValidURL(req.URL) {
		http.Error(w, "url must be a valid absolute http(s) URL", http.StatusBadRequest)
		return
	}

	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	code, err := h.store.CreateLink(r.Context(), req.URL, userID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	resp := createLinkResponse{
		ShortCode: code,
		ShortURL:  baseURL + "/" + code,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) Redirect(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")

	longURL, err := h.store.GetLongURL(r.Context(), code)
	if errors.Is(err, ErrNotFound) {
		http.Error(w, "short link not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	h.clicks.Log(ClickEvent{
		ShortCode: code,
		IPAddress: clientIP(r),
		UserAgent: r.UserAgent(),
		Referrer:  r.Referer(),
		Timestamp: time.Now(),
	})

	http.Redirect(w, r, longURL, http.StatusFound)
}

func clientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return strings.TrimSpace(strings.Split(fwd, ",")[0])
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func (h *Handler) ListMyLinks(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	links, err := h.store.ListByOwner(r.Context(), userID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(links)
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) AnalyticsSummary(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")

	if _, err := h.store.GetLongURL(r.Context(), code); err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "short link not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	days := 30
	if raw := r.URL.Query().Get("days"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 && n <= 90 {
			days = n
		}
	}

	summary, err := h.analytics.Summary(r.Context(), code, days)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)

}

func isValidURL(raw string) bool {
	parsed, err := url.ParseRequestURI(raw)
	if err != nil {
		return false
	}

	return parsed.Scheme == "http" || parsed.Scheme == "https"
}
