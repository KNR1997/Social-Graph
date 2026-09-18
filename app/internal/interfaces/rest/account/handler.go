// Package account is the HTTP adapter for the account aggregate. It translates
// JSON into use-case calls and account sentinels into status codes, and
// contains no business rules.
//
// It imports exactly one domain package. Anything a second resource would also
// need belongs in rest/httpx, phrased in transport terms.
package account

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/kethaka-creskit/go-ddd-service/internal/domain/account/entity"
	"github.com/kethaka-creskit/go-ddd-service/internal/interfaces/rest/httpx"
)

// Service is the port this handler needs, declared here on the consuming side
// rather than exported by the application package. That keeps the dependency
// narrow and lets the handler tests use a stub with no database.
type Service interface {
	OpenAccount(ctx context.Context, owner, currency string) (*entity.Account, error)
	GetAccount(ctx context.Context, id uuid.UUID) (*entity.Account, error)
	ListAccounts(ctx context.Context, page entity.Page) ([]*entity.Account, error)
	Deposit(ctx context.Context, id uuid.UUID, minor int64, currency string) (*entity.Account, error)
	Withdraw(ctx context.Context, id uuid.UUID, minor int64, currency string) (*entity.Account, error)
	CloseAccount(ctx context.Context, id uuid.UUID) (*entity.Account, error)
}

// Handler serves the account endpoints.
type Handler struct {
	accounts Service
	log      *slog.Logger
}

// NewHandler builds the handler.
func NewHandler(accounts Service, log *slog.Logger) *Handler {
	return &Handler{accounts: accounts, log: log}
}

// Routes mounts the account endpoints on their own sub-router, so the top-level
// router does not accumulate every path in the service.
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Post("/", h.open)
	r.Get("/", h.list)
	r.Route("/{accountID}", func(r chi.Router) {
		r.Get("/", h.get)
		r.Post("/deposit", h.deposit)
		r.Post("/withdraw", h.withdraw)
		r.Delete("/", h.close)
	})

	return r
}

// fail writes an error response, binding this package's sentinel mapping to the
// shared writer so every call site stays one line.
func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	httpx.WriteError(w, r, h.log, err, statusFor)
}

func (h *Handler) open(w http.ResponseWriter, r *http.Request) {
	req, err := httpx.Decode[openAccountRequest](r)
	if err != nil {
		h.fail(w, r, err)

		return
	}

	acc, err := h.accounts.OpenAccount(r.Context(), req.Owner, req.Currency)
	if err != nil {
		h.fail(w, r, err)

		return
	}

	w.Header().Set("Location", "/v1/accounts/"+acc.ID().String())
	httpx.WriteJSON(w, h.log, http.StatusCreated, toResponse(acc))
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, err := accountID(r)
	if err != nil {
		h.fail(w, r, err)

		return
	}

	acc, err := h.accounts.GetAccount(r.Context(), id)
	if err != nil {
		h.fail(w, r, err)

		return
	}

	httpx.WriteJSON(w, h.log, http.StatusOK, toResponse(acc))
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	p := httpx.ParsePage(r)
	page := entity.Page{Limit: p.Limit, Offset: p.Offset}

	accounts, err := h.accounts.ListAccounts(r.Context(), page)
	if err != nil {
		h.fail(w, r, err)

		return
	}

	httpx.WriteJSON(w, h.log, http.StatusOK, listResponse{
		Accounts: toResponses(accounts),
		Limit:    page.Limit,
		Offset:   page.Offset,
	})
}

func (h *Handler) deposit(w http.ResponseWriter, r *http.Request) {
	h.changeBalance(w, r, h.accounts.Deposit)
}

func (h *Handler) withdraw(w http.ResponseWriter, r *http.Request) {
	h.changeBalance(w, r, h.accounts.Withdraw)
}

// changeBalance is the shared shape of the two balance endpoints.
func (h *Handler) changeBalance(
	w http.ResponseWriter,
	r *http.Request,
	apply func(context.Context, uuid.UUID, int64, string) (*entity.Account, error),
) {
	id, err := accountID(r)
	if err != nil {
		h.fail(w, r, err)

		return
	}

	req, err := httpx.Decode[amountRequest](r)
	if err != nil {
		h.fail(w, r, err)

		return
	}

	if req.AmountMinor == nil {
		h.fail(w, r, httpx.NewRequestError("amount_minor is required"))

		return
	}

	acc, err := apply(r.Context(), id, *req.AmountMinor, req.Currency)
	if err != nil {
		h.fail(w, r, err)

		return
	}

	httpx.WriteJSON(w, h.log, http.StatusOK, toResponse(acc))
}

func (h *Handler) close(w http.ResponseWriter, r *http.Request) {
	id, err := accountID(r)
	if err != nil {
		h.fail(w, r, err)

		return
	}

	acc, err := h.accounts.CloseAccount(r.Context(), id)
	if err != nil {
		h.fail(w, r, err)

		return
	}

	httpx.WriteJSON(w, h.log, http.StatusOK, toResponse(acc))
}

// accountID pulls the path parameter and validates it.
func accountID(r *http.Request) (uuid.UUID, error) {
	raw := chi.URLParam(r, "accountID")

	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, httpx.NewRequestError("%q is not a valid account id", raw)
	}

	return id, nil
}
