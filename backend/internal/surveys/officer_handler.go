package surveys

import (
	"encoding/csv"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/cageymage/fuzion/backend/internal/auth"
)

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "body: must be valid JSON"})
		return
	}

	officer, _ := auth.UserFrom(r.Context())
	created, err := h.service.Create(r.Context(), officer, req)
	if err != nil {
		h.writeError(w, r, "create survey", err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "body: must be valid JSON"})
		return
	}

	updated, err := h.service.Update(r.Context(), chi.URLParam(r, "id"), req)
	if err != nil {
		h.writeError(w, r, "update survey", err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *Handler) addOption(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	var req AddOptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "body: must be valid JSON"})
		return
	}

	updated, err := h.service.AddOption(r.Context(), chi.URLParam(r, "id"), chi.URLParam(r, "questionId"), req)
	if err != nil {
		h.writeError(w, r, "add survey option", err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *Handler) listResponses(w http.ResponseWriter, r *http.Request) {
	responses, err := h.service.Responses(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		h.writeError(w, r, "list survey responses", err)
		return
	}
	writeJSON(w, http.StatusOK, responses)
}

func (h *Handler) voidResponse(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	var req VoidRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "body: must be valid JSON"})
		return
	}

	officer, _ := auth.UserFrom(r.Context())
	response, err := h.service.Void(r.Context(), chi.URLParam(r, "id"), chi.URLParam(r, "responseId"), officer, req)
	if err != nil {
		h.writeError(w, r, "void survey response", err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) exportResponses(w http.ResponseWriter, r *http.Request) {
	export, err := h.service.Export(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		h.writeError(w, r, "export survey responses", err)
		return
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="survey-`+export.SurveyID.String()+`-responses.csv"`)
	if err := csv.NewWriter(w).WriteAll(export.Rows); err != nil {
		slog.ErrorContext(r.Context(), "write survey responses csv", "error", err)
	}
}
