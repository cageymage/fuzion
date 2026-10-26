package auth

import "context"

type Identity struct {
	ProviderUserID string
	Username       string
	AvatarURL      string

	// InDiscordServer is only ever set by the Discord provider, and only when it was given a guild to look for.
	InDiscordServer bool

	// Characters is only ever set by the Battle.net provider: the WoW characters on the account.
	Characters []AccountCharacter
}

type AccountCharacter struct {
	Name      string
	RealmSlug string
}

type Provider interface {
	AuthURL(state string) string
	Exchange(ctx context.Context, code string) (Identity, error)
}
