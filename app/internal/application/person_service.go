package application

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/kethaka-creskit/go-ddd-service/internal/domain/person/entity"
)

type PersonService struct {
	personRepository entity.Repository
	tx               TxManager
	log              *slog.Logger
	clock            Clock
	ids              IDGenerator
}

func NewPersonService(
	personRepository entity.Repository,
	tx TxManager,
	clock Clock,
	ids IDGenerator,
	log *slog.Logger,
) *PersonService {
	return &PersonService{
		personRepository: personRepository,
		tx:               tx,
		log:              log,
		clock:            clock,
		ids:              ids,
	}
}

func (s *PersonService) CreatePerson(
	ctx context.Context,
	userID uuid.UUID,
	isSelf bool,
	details entity.Details,
) (*entity.Person, error) {
	s.log.InfoContext(ctx, "creating person",
		slog.String("user_id", userID.String()),
	)

	person, err := entity.Create(
		s.ids.NewID(),
		userID,
		isSelf,
		details,
		s.clock.Now(),
	)
	if err != nil {
		return nil, fmt.Errorf("application - CreatePerson - entity.Create: %w", err)
	}

	if err := s.personRepository.Add(ctx, person); err != nil {
		return nil, fmt.Errorf("application - CreatePerson - personRepository.Add: %w", err)
	}

	s.log.InfoContext(ctx, "person created",
		slog.String("person_id", person.ID().String()),
	)

	return person, nil
}

func (s *PersonService) ListPersons(
	ctx context.Context,
	userID uuid.UUID,
	filter entity.Filter,
) ([]*entity.Person, int64, error) {
	persons, err := s.personRepository.List(ctx, userID, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("application - ListPersons - personRepository.List: %w", err)
	}

	total, err := s.personRepository.Count(ctx, userID, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("application - ListPersons - personRepository.Count: %w", err)
	}

	return persons, total, nil
}

func (s *PersonService) GetPerson(
	ctx context.Context,
	userID, id uuid.UUID,
) (*entity.Person, error) {
	person, err := s.personRepository.ByID(ctx, userID, id)
	if err != nil {
		return nil, fmt.Errorf("application - GetPerson - personRepository.ByID: %w", err)
	}

	return person, nil
}

func (s *PersonService) UpdatePerson(
	ctx context.Context,
	userID, id uuid.UUID,
	patch entity.DetailsPatch,
) (*entity.Person, error) {
	var updated *entity.Person

	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		person, err := s.personRepository.ByID(ctx, userID, id)
		if err != nil {
			return fmt.Errorf("personRepository.ByID: %w", err)
		}

		if err := person.ApplyPatch(patch, s.clock.Now()); err != nil {
			return fmt.Errorf("person.ApplyPatch: %w", err)
		}

		if err := s.personRepository.Update(ctx, person); err != nil {
			return fmt.Errorf("personRepository.Update: %w", err)
		}

		updated = person

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("application - UpdatePerson - %w", err)
	}

	return updated, nil
}

func (s *PersonService) DeletePerson(ctx context.Context, userID, id uuid.UUID) error {
	if err := s.personRepository.Delete(ctx, userID, id); err != nil {
		return fmt.Errorf("application - DeletePerson - personRepository.Delete: %w", err)
	}

	s.log.InfoContext(ctx, "person deleted",
		slog.String("person_id", id.String()),
	)

	return nil
}
