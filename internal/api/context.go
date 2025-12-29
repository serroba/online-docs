package api

import "context"

type contextKey string

const userIDKey contextKey = "userID"

// UserIDFromContext extracts the user ID from the context.
// Returns empty string if not present.
func UserIDFromContext(ctx context.Context) string {
	if v := ctx.Value(userIDKey); v != nil {
		if userID, ok := v.(string); ok {
			return userID
		}
	}

	return ""
}

// withUserID returns a new context with the user ID set.
func withUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}
