package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"time"
)

const (
	SessionCookieName = "fuzion_session"
	stateCookieName   = "fuzion_oauth_state"
	stateTTL          = 10 * time.Minute
)

func newSessionToken() (string, error) {
	return randomHex(32)
}

func newState() (string, error) {
	return randomHex(16)
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate random token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func sessionCookie(r *http.Request, token string) *http.Cookie {
	return newCookie(r, SessionCookieName, token, int(sessionTTL/time.Second))
}

func stateCookie(r *http.Request, state string) *http.Cookie {
	return newCookie(r, stateCookieName, state, int(stateTTL/time.Second))
}

func expiredCookie(r *http.Request, name string) *http.Cookie {
	return newCookie(r, name, "", -1)
}

func newCookie(r *http.Request, name, value string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   !isLocalhost(r),
		SameSite: http.SameSiteLaxMode,
	}
}

// Browsers drop Secure cookies over plain http, which is what local dev
// (Vite proxy -> 127.0.0.1) and httptest servers use.
func isLocalhost(r *http.Request) bool {
	host := r.Host
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
