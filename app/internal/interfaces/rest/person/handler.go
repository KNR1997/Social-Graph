package person

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/kethaka-creskit/go-ddd-service/internal/domain/person/entity"
	"github.com/kethaka-creskit/go-ddd-service/internal/interfaces/rest/httpx"
)

var errNoOwner = errors.New("no authenticated owner on the request")

type Service interface {
	CreatePerson(
		ctx context.Context, userID uuid.UUID, isSelf bool, details entity.Details,
	) (*entity.Person, error)
	GetPerson(ctx context.Context, userID, id uuid.UUID) (*entity.Person, error)
	ListPersons(
		ctx context.Context, userID uuid.UUID, filter entity.Filter,
	) ([]*entity.Person, int64, error)
	UpdatePerson(
		ctx context.Context, userID, id uuid.UUID, patch entity.DetailsPatch,
	) (*entity.Person, error)
	DeletePerson(ctx context.Context, userID, id uuid.UUID) error
}

type Handler struct {
	people Service
	log    *slog.Logger
}

func NewHandler(people Service, log *slog.Logger) *Handler {
	return &Handler{people: people, log: log}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Post("/", h.create)
	r.Get("/", h.list)
	r.Route("/{personID}", func(r chi.Router) {
		r.Get("/", h.get)
		r.Patch("/", h.update)
		r.Delete("/", h.delete)
	})

	return r
}

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	httpx.WriteError(w, r, h.log, err, statusFor)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	owner, err := ownerID(r)
	if err != nil {
		h.fail(w, r, err)

		return
	}

	req, err := httpx.Decode[createPersonRequest](r)
	if err != nil {
		h.fail(w, r, err)

		return
	}

	created, err := h.people.CreatePerson(r.Context(), owner, false, req.toDetails())
	if err != nil {
		h.fail(w, r, err)

		return
	}

	w.Header().Set("Location", "/v1/people/"+created.ID().String())
	httpx.WriteJSON(w, h.log, http.StatusCreated, toResponse(created))
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	owner, id, err := ownerAndPerson(r)
	if err != nil {
		h.fail(w, r, err)

		return
	}

	found, err := h.people.GetPerson(r.Context(), owner, id)
	if err != nil {
		h.fail(w, r, err)

		return
	}

	httpx.WriteJSON(w, h.log, http.StatusOK, toResponse(found))
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	owner, err := ownerID(r)
	if err != nil {
		h.fail(w, r, err)

		return
	}

	p := httpx.ParsePage(r)
	filter := entity.Filter{
		Page:   entity.Page{Limit: p.Limit, Offset: p.Offset},
		Search: r.URL.Query().Get("q"),
	}

	people, total, err := h.people.ListPersons(r.Context(), owner, filter)
	if err != nil {
		h.fail(w, r, err)

		return
	}

	httpx.WriteJSON(w, h.log, http.StatusOK, listResponse{
		People: toResponses(people),
		Total:  total,
		Limit:  filter.Page.Limit,
		Offset: filter.Page.Offset,
	})
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	owner, id, err := ownerAndPerson(r)
	if err != nil {
		h.fail(w, r, err)

		return
	}

	req, err := httpx.Decode[updatePersonRequest](r)
	if err != nil {
		h.fail(w, r, err)

		return
	}

	updated, err := h.people.UpdatePerson(r.Context(), owner, id, req.toPatch())
	if err != nil {
		h.fail(w, r, err)

		return
	}

	httpx.WriteJSON(w, h.log, http.StatusOK, toResponse(updated))
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	owner, id, err := ownerAndPerson(r)
	if err != nil {
		h.fail(w, r, err)

		return
	}

	if err := h.people.DeletePerson(r.Context(), owner, id); err != nil {
		h.fail(w, r, err)

		return
	}

	httpx.WriteJSON(w, h.log, http.StatusNoContent, nil)
}

func ownerID(r *http.Request) (uuid.UUID, error) {
	owner, ok := httpx.OwnerFrom(r.Context())
	if !ok {
		return uuid.Nil, errNoOwner
	}

	return owner, nil
}

func personID(r *http.Request) (uuid.UUID, error) {
	raw := chi.URLParam(r, "personID")

	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, httpx.NewRequestError("%q is not a valid person id", raw)
	}

	return id, nil
}

func ownerAndPerson(r *http.Request) (owner, id uuid.UUID, err error) {
	if owner, err = ownerID(r); err != nil {
		return uuid.Nil, uuid.Nil, err
	}

	if id, err = personID(r); err != nil {
		return uuid.Nil, uuid.Nil, err
	}

	return owner, id, nil
}
