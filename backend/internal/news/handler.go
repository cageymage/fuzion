package news

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

const (
	defaultLimit = 10
	maxLimit     = 50
)

var categories = []string{"raid-progress", "recruitment", "guild-news"}

type embeddedPosts struct {
	News []Post `json:"news"`
}

type listResponse struct {
	Total    int             `json:"total"`
	Links    map[string]link `json:"_links"`
	Embedded embeddedPosts   `json:"_embedded"`
}

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Register(r chi.Router) {
	r.Get("/news", h.listPosts)
}

func (h *Handler) listPosts(w http.ResponseWriter, r *http.Request) {
	params, err := parseListParams(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	page, err := h.service.ListPosts(r.Context(), params)
	if err != nil {
		slog.ErrorContext(r.Context(), "list news posts", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "news posts could not be loaded"})
		return
	}
	writeJSON(w, http.StatusOK, listResponse{
		Total:    page.Total,
		Links:    pageLinks(r.URL.Path, params, page.Total, r.URL.Query().Has("limit")),
		Embedded: embeddedPosts{News: page.Posts},
	})
}

func parseListParams(r *http.Request) (ListParams, error) {
	limit, err := parseLimit(r)
	if err != nil {
		return ListParams{}, err
	}
	offset, err := parseOffset(r)
	if err != nil {
		return ListParams{}, err
	}
	category, err := parseCategory(r)
	if err != nil {
		return ListParams{}, err
	}
	return ListParams{Limit: limit, Offset: offset, Category: category}, nil
}

func parseLimit(r *http.Request) (int, error) {
	if !r.URL.Query().Has("limit") {
		return defaultLimit, nil
	}
	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil || limit < 1 || limit > maxLimit {
		return 0, errors.New("limit: must be an integer between 1 and 50")
	}
	return limit, nil
}

func parseOffset(r *http.Request) (int, error) {
	if !r.URL.Query().Has("offset") {
		return 0, nil
	}
	offset, err := strconv.Atoi(r.URL.Query().Get("offset"))
	if err != nil || offset < 0 {
		return 0, errors.New("offset: must be a non-negative integer")
	}
	return offset, nil
}

func parseCategory(r *http.Request) (string, error) {
	if !r.URL.Query().Has("category") {
		return "", nil
	}
	category := r.URL.Query().Get("category")
	if !slices.Contains(categories, category) {
		return "", errors.New("category: must be one of " + strings.Join(categories, ", "))
	}
	return category, nil
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("encode news response", "error", err)
	}
}
