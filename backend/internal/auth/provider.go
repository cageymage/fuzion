package auth

import "context"

type Identity struct {
	ProviderUserID string
	Username       string
	AvatarURL      string
}

type Provider interface {
	AuthURL(state string) string
	Exchange(ctx context.Context, code string) (Identity, error)
}
