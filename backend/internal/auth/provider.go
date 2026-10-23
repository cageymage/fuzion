package auth

import "context"

type Identity struct {
	ProviderUserID string
	Username       string
	AvatarURL      string

	// InDiscordServer is only ever set by the Discord provider, and only when it was given a guild to look for.
	InDiscordServer bool
}

type Provider interface {
	AuthURL(state string) string
	Exchange(ctx context.Context, code string) (Identity, error)
}
