package firebase

import (
	"context"
	"fmt"

	fb "firebase.google.com/go/v4"
	fbauth "firebase.google.com/go/v4/auth"
	"google.golang.org/api/option"
)

// TokenInfo holds the user information extracted from a verified Firebase ID token.
type TokenInfo struct {
	UID         string
	Email       string
	PhoneNumber string
	DisplayName string
	PhotoURL    string
	Provider    string // sign-in provider: "google.com", "phone", "facebook.com", etc.
}

// Verifier verifies Firebase ID tokens.
type Verifier interface {
	VerifyIDToken(ctx context.Context, idToken string) (*TokenInfo, error)
}

type firebaseVerifier struct {
	client *fbauth.Client
}

// NewVerifier initialises a Firebase Admin SDK auth client and returns a Verifier.
// credentialsFile is the path to the service-account JSON. If empty, the SDK falls
// back to GOOGLE_APPLICATION_CREDENTIALS or the default compute credentials.
func NewVerifier(ctx context.Context, credentialsFile, projectID string) (Verifier, error) {
	var opts []option.ClientOption
	if credentialsFile != "" {
		opts = append(opts, option.WithCredentialsFile(credentialsFile))
	}

	cfg := &fb.Config{}
	if projectID != "" {
		cfg.ProjectID = projectID
	}

	app, err := fb.NewApp(ctx, cfg, opts...)
	if err != nil {
		return nil, fmt.Errorf("firebase: failed to initialise app: %w", err)
	}

	client, err := app.Auth(ctx)
	if err != nil {
		return nil, fmt.Errorf("firebase: failed to get auth client: %w", err)
	}

	return &firebaseVerifier{client: client}, nil
}

func (v *firebaseVerifier) VerifyIDToken(ctx context.Context, idToken string) (*TokenInfo, error) {
	token, err := v.client.VerifyIDToken(ctx, idToken)
	if err != nil {
		return nil, fmt.Errorf("firebase: invalid id token: %w", err)
	}

	info := &TokenInfo{
		UID: token.UID,
	}

	// Extract claims populated by Firebase.
	if email, ok := token.Claims["email"].(string); ok {
		info.Email = email
	}
	if phone, ok := token.Claims["phone_number"].(string); ok {
		info.PhoneNumber = phone
	}
	if name, ok := token.Claims["name"].(string); ok {
		info.DisplayName = name
	}
	if pic, ok := token.Claims["picture"].(string); ok {
		info.PhotoURL = pic
	}

	// Determine the sign-in provider from the Firebase token.
	if provider, ok := token.Claims["firebase"].(map[string]interface{}); ok {
		if signIn, ok := provider["sign_in_provider"].(string); ok {
			info.Provider = signIn
		}
	}

	return info, nil
}