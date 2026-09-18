package auth

import (
	"time"

	"github.com/google/uuid"

	"github.com/kethaka-creskit/go-ddd-service/internal/domain/auth/entity"
)

// registerRequest is the POST /v1/auth/register body.
type registerRequest struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

// loginRequest is the POST /v1/auth/login body.
type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// renameRequest is the PATCH /v1/auth/me body.
type renameRequest struct {
	Name string `json:"name"`
}

// changePasswordRequest is the POST /v1/auth/me/password body.
type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// userResponse is the wire representation of the signed-in user.
//
// There is no password field, and there is no variant of this type that has
// one. The aggregate's digest is reachable only through a method the mapper
// below does not call, so "we forgot to strip the hash" is not a bug this
// endpoint can develop.
type userResponse struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// sessionResponse is what a successful sign-in returns alongside the cookie.
//
// It deliberately does not carry the token. The token is in the Set-Cookie
// header and nowhere else: putting it in a JSON body would hand it to any
// script on the page, which is exactly what HttpOnly is for.
type sessionResponse struct {
	User      userResponse `json:"user"`
	ExpiresAt time.Time    `json:"expires_at"`
}

func toResponse(u *entity.User) userResponse {
	return userResponse{
		ID:        u.ID(),
		Email:     u.Email().String(),
		Name:      u.Name(),
		Role:      string(u.Role()),
		CreatedAt: u.CreatedAt(),
		UpdatedAt: u.UpdatedAt(),
	}
}

func toSessionResponse(u *entity.User, cred entity.Credential) sessionResponse {
	return sessionResponse{
		User:      toResponse(u),
		ExpiresAt: cred.Session.ExpiresAt(),
	}
}
