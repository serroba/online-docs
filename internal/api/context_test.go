package api_test

import (
	"context"
	"testing"

	"github.com/serroba/online-docs/internal/api"
)

func TestUserIDFromContext(t *testing.T) {
	t.Parallel()

	t.Run("returns empty string when not set", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		userID := api.UserIDFromContext(ctx)

		if userID != "" {
			t.Errorf("expected empty string, got %q", userID)
		}
	})
}
