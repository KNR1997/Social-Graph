package account

import (
	"errors"
	"net/http"

	"github.com/kethaka-creskit/go-ddd-service/internal/domain/account/entity"
	"github.com/kethaka-creskit/go-ddd-service/internal/interfaces/rest/httpx"
)

// Machine-readable error codes specific to accounts. Clients branch on these,
// so they are part of the API contract: add new ones freely, never reword an
// existing one. The resource-independent codes live in httpx.
const (
	codeNotFound       = "account_not_found"
	codeConflict       = "concurrent_modification"
	codeInsufficient   = "insufficient_funds"
	codeNotActive      = "account_not_active"
	codeBalanceNotZero = "balance_not_zero"
)

// statusFor maps an account sentinel to a status, a stable machine-readable
// code, and a client-facing message. It reports false for anything that is not
// an account error, leaving httpx to fall through to a 500.
//
// The message is written out per case instead of being derived from the error
// text. That keeps the wire contract stable when an internal message is
// reworded, and it guarantees that no wrapped internal detail (a table name, or
// a balance the caller may not be entitled to see) escapes. The full error
// still goes to the log.
//
// This function is the only place in this package that knows both the domain
// vocabulary and the HTTP one.
func statusFor(err error) (httpx.Status, bool) {
	switch {
	case errors.Is(err, entity.ErrNotFound):
		return status(http.StatusNotFound, codeNotFound, "account not found"), true

	case errors.Is(err, entity.ErrConflict):
		return status(http.StatusConflict, codeConflict,
			"the account was modified concurrently, retry the request"), true

	case errors.Is(err, entity.ErrInsufficientFunds):
		return status(http.StatusUnprocessableEntity, codeInsufficient, "insufficient funds"), true

	case errors.Is(err, entity.ErrNotActive):
		return status(http.StatusConflict, codeNotActive, "the account is not active"), true

	case errors.Is(err, entity.ErrBalanceNotZero):
		return status(http.StatusConflict, codeBalanceNotZero,
			"the account balance must be zero before it can be closed"), true

	case errors.Is(err, entity.ErrEmptyOwner):
		return badRequest("owner must not be empty"), true

	case errors.Is(err, entity.ErrInvalidCurrency):
		return badRequest("currency must be a 3-letter ISO-4217 code"), true

	case errors.Is(err, entity.ErrCurrencyMismatch):
		return badRequest("the amount currency does not match the account currency"), true

	case errors.Is(err, entity.ErrNotPositive):
		return badRequest("amount must be positive"), true

	case errors.Is(err, entity.ErrInvalidStatus):
		return badRequest("invalid account status"), true

	default:
		return httpx.Status{}, false
	}
}

func status(code int, slug, message string) httpx.Status {
	return httpx.Status{HTTP: code, Code: slug, Message: message}
}

func badRequest(message string) httpx.Status {
	return status(http.StatusBadRequest, httpx.CodeInvalidRequest, message)
}
