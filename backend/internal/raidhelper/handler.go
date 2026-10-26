package raidhelper

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Raid-Helper names the webhook type only in this header, which its docs do not
// mention; the body has no type field.
const eventTypeHeader = "X-Raidhelper-Event"

// A real event body is about 21 KB; the limit leaves room for large signup lists.
const maxBodyBytes = 1 << 20

// Raid-Helper's webhook settings cover the whole Discord server, with no way to pick
// a channel, so every event arrives here and the channel decides what happens to it.
const (
	foreverRaidPlanningChannelID = "1270113143124660246"
	retailRaidPlanningChannelID  = "1281325821813985391"
)

type Handler struct {
	service    *Service
	webhookKey string
}

// An empty webhookKey disables the endpoint: it is not registered, so requests get a 404.
func NewHandler(service *Service, webhookKey string) *Handler {
	return &Handler{service: service, webhookKey: webhookKey}
}

func (h *Handler) Register(r chi.Router) {
	if h.webhookKey == "" {
		return
	}
	r.Post("/webhooks/raid-helper", h.receive)
}

func (h *Handler) receive(w http.ResponseWriter, r *http.Request) {
	// Raid-Helper sends the key with no prefix. Checking it is optional on its side,
	// so this is the only thing stopping anyone who knows the URL from posting events.
	if subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), []byte(h.webhookKey)) != 1 {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid webhook key"})
		return
	}

	var apply func(context.Context, Event) error
	switch eventType := r.Header.Get(eventTypeHeader); eventType {
	case "webhookeventcreate", "webhookeventupdate":
		apply = h.service.Save
	case "webhookeventdelete":
		apply = h.service.Delete
	default:
		slog.InfoContext(r.Context(), "raid-helper webhook ignored", "type", eventType)
		writeJSON(w, http.StatusOK, map[string]string{"status": "ignored"})
		return
	}

	var event Event
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes)).Decode(&event); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "body is not a raid-helper event"})
		return
	}

	// Ignored events still get a 200 so Raid-Helper does not retry them.
	switch event.ChannelID {
	case foreverRaidPlanningChannelID:
	case retailRaidPlanningChannelID:
		writeJSON(w, http.StatusOK, map[string]string{"status": "ignored"})
		return
	default:
		slog.InfoContext(r.Context(), "raid-helper webhook from unknown channel ignored", "channelId", event.ChannelID, "channelName", event.ChannelName)
		writeJSON(w, http.StatusOK, map[string]string{"status": "ignored"})
		return
	}

	if err := apply(r.Context(), event); err != nil {
		if errors.Is(err, ErrInvalidEvent) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		slog.ErrorContext(r.Context(), "apply raid-helper webhook", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "event could not be stored"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("encode raid-helper webhook response", "error", err)
	}
}
