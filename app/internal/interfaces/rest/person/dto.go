package person

import (
	"time"

	"github.com/google/uuid"

	"github.com/kethaka-creskit/go-ddd-service/internal/domain/person/entity"
)

type createPersonRequest struct {
	FullName     string `json:"full_name"`
	Email        string `json:"email"`
	Phone        string `json:"phone"`
	Occupation   string `json:"occupation"`
	Organization string `json:"organization"`
	Location     string `json:"location"`
	Notes        string `json:"notes"`
}

func (r *createPersonRequest) toDetails() entity.Details {
	return entity.Details{
		FullName:     r.FullName,
		Email:        r.Email,
		Phone:        r.Phone,
		Occupation:   r.Occupation,
		Organization: r.Organization,
		Location:     r.Location,
		Notes:        r.Notes,
	}
}

type updatePersonRequest struct {
	FullName     *string `json:"full_name"`
	Email        *string `json:"email"`
	Phone        *string `json:"phone"`
	Occupation   *string `json:"occupation"`
	Organization *string `json:"organization"`
	Location     *string `json:"location"`
	Notes        *string `json:"notes"`
}

func (r *updatePersonRequest) toPatch() entity.DetailsPatch {
	return entity.DetailsPatch{
		FullName:     r.FullName,
		Email:        r.Email,
		Phone:        r.Phone,
		Occupation:   r.Occupation,
		Organization: r.Organization,
		Location:     r.Location,
		Notes:        r.Notes,
	}
}

type personResponse struct {
	ID           uuid.UUID `json:"id"`
	IsSelf       bool      `json:"is_self"`
	FullName     string    `json:"full_name"`
	Email        string    `json:"email"`
	Phone        string    `json:"phone"`
	Occupation   string    `json:"occupation"`
	Organization string    `json:"organization"`
	Location     string    `json:"location"`
	Notes        string    `json:"notes"`
	Version      int64     `json:"version"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type listResponse struct {
	People []personResponse `json:"people"`
	Total  int64            `json:"total"`
	Limit  int32            `json:"limit"`
	Offset int32            `json:"offset"`
}

func toResponse(p *entity.Person) personResponse {
	d := p.Details()

	return personResponse{
		ID:           p.ID(),
		IsSelf:       p.IsSelf(),
		FullName:     d.FullName,
		Email:        d.Email,
		Phone:        d.Phone,
		Occupation:   d.Occupation,
		Organization: d.Organization,
		Location:     d.Location,
		Notes:        d.Notes,
		Version:      p.Version(),
		CreatedAt:    p.CreatedAt(),
		UpdatedAt:    p.UpdatedAt(),
	}
}

func toResponses(people []*entity.Person) []personResponse {
	out := make([]personResponse, 0, len(people))
	for _, p := range people {
		out = append(out, toResponse(p))
	}

	return out
}
