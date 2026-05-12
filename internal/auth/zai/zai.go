package zai

import (
	"context"
	"fmt"

	cliproxyauth "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/auth"
)

type ZaiAuth struct {
}

func NewZaiAuth() *ZaiAuth {
	return &ZaiAuth{}
}

// FetchGuestToken performs a simple request to chat.z.ai and looks for the generated JWT token in the response headers.
// However, since the token is actually generated via a background call or injected, we simulate the logic here
// or require the user to provide it.
func (z *ZaiAuth) FetchGuestToken(ctx context.Context) (string, error) {
	// For actual dynamic fetch without browser, Z.ai might require JS execution.
	// As a fallback, we return an error indicating manual setup or rely on a predefined token in config.
	return "", fmt.Errorf("dynamic guest token fetch requires JS execution. Please provide a manual token in auth file.")
}

// ZaiCreds extracts the token from the auth store.
func ZaiCreds(a *cliproxyauth.Auth) string {
	if a == nil {
		return ""
	}
	// Fallback to extract from AccountInfo
	_, info := a.AccountInfo()
	if info != "" {
		return "Bearer " + info
	}
	return ""
}
