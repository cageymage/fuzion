package turnstile_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/cageymage/fuzion/backend/internal/testutil"
	"github.com/cageymage/fuzion/backend/internal/turnstile"
)

func newClient(fake *testutil.FakeTurnstile, secret string) *turnstile.Client {
	return turnstile.NewClient(turnstile.Config{SecretKey: secret, BaseURL: fake.URL}, fake.Client())
}

func TestVerify_ReturnsNil_WhenCloudflareAcceptsTheToken(t *testing.T) {
	// given a fake Cloudflare that accepts the token
	fake := testutil.NewFakeTurnstile(t)

	// when I verify it
	err := newClient(fake, testutil.TurnstileSecretKey).Verify(context.Background(), testutil.ValidTurnstileToken, "203.0.113.7")

	// then I expect no error
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
}

func TestVerify_ReturnsErrInvalidToken_WhenCloudflareRejectsTheToken(t *testing.T) {
	// given a fake Cloudflare and a token it does not know
	fake := testutil.NewFakeTurnstile(t)

	// when I verify the token
	err := newClient(fake, testutil.TurnstileSecretKey).Verify(context.Background(), "forged", "203.0.113.7")

	// then I expect ErrInvalidToken
	if !errors.Is(err, turnstile.ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestVerify_ReturnsUnexpectedStatusError_WhenCloudflareAnswersServerError(t *testing.T) {
	// given a Cloudflare that answers 500
	fake := testutil.NewFakeTurnstile(t)
	fake.Status = 500

	// when I verify a token
	err := newClient(fake, testutil.TurnstileSecretKey).Verify(context.Background(), testutil.ValidTurnstileToken, "")

	// then I expect an error that is not ErrInvalidToken and does not leak the secret
	if err == nil || errors.Is(err, turnstile.ErrInvalidToken) {
		t.Fatalf("expected a transport-style error, got %v", err)
	}
	if strings.Contains(err.Error(), testutil.TurnstileSecretKey) {
		t.Errorf("error leaks the secret: %v", err)
	}
}
