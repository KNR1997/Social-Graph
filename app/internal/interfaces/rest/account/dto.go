package account

import (
	"time"

	"github.com/google/uuid"

	"github.com/kethaka-creskit/go-ddd-service/internal/domain/account/entity"
)

// openAccountRequest is the POST /v1/accounts body.
type openAccountRequest struct {
	Owner    string `json:"owner"`
	Currency string `json:"currency"`
}

// amountRequest is the deposit and withdraw body. AmountMinor is a pointer so a
// missing field is distinguishable from an explicit zero, which the domain
// rejects for a different reason and with a different message.
type amountRequest struct {
	AmountMinor *int64 `json:"amount_minor"`
	Currency    string `json:"currency"`
}

// accountResponse is the wire representation of an account. It is a separate
// type from the aggregate on purpose: the API shape can stay stable while the
// aggregate evolves, and the aggregate never has to grow json tags.
type accountResponse struct {
	ID           uuid.UUID `json:"id"`
	Owner        string    `json:"owner"`
	BalanceMinor int64     `json:"balance_minor"`
	Currency     string    `json:"currency"`
	Status       string    `json:"status"`
	Version      int64     `json:"version"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// listResponse wraps a page of accounts in an object rather than returning a
// bare array, so pagination metadata can be added later without a breaking change.
type listResponse struct {
	Accounts []accountResponse `json:"accounts"`
	Limit    int32             `json:"limit"`
	Offset   int32             `json:"offset"`
}

func toResponse(a *entity.Account) accountResponse {
	return accountResponse{
		ID:           a.ID(),
		Owner:        a.Owner(),
		BalanceMinor: a.Balance().Minor(),
		Currency:     string(a.Balance().Currency()),
		Status:       string(a.Status()),
		Version:      a.Version(),
		CreatedAt:    a.CreatedAt(),
		UpdatedAt:    a.UpdatedAt(),
	}
}

func toResponses(accounts []*entity.Account) []accountResponse {
	out := make([]accountResponse, 0, len(accounts))
	for _, a := range accounts {
		out = append(out, toResponse(a))
	}

	return out
}
