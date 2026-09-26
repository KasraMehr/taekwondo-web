package athletes

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

var ErrInvalidAthlete = errors.New("invalid athlete data")

type Service struct {
	repository *Repository
}

func NewService(repository *Repository) *Service {
	return &Service{
		repository: repository,
	}
}

func (s *Service) Create(
	ctx context.Context,
	request CreateAthleteRequest,
) (*Athlete, error) {
	request.Name = strings.TrimSpace(request.Name)

	if request.Name == "" {
		return nil, ErrInvalidAthlete
	}

	birthDate, err := parseBirthDate(request.BirthDate)
	if err != nil {
		return nil, err
	}

	athlete := &Athlete{
		ID:           uuid.New(),
		Name:         request.Name,
		NationalCode: request.NationalCode,
		BirthDate:    birthDate,
		Gender:       request.Gender,
		Phone:        request.Phone,
		Email:        request.Email,
		IsActive:     true,
	}

	if err := s.repository.Create(ctx, athlete); err != nil {
		return nil, err
	}

	return athlete, nil
}

func (s *Service) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (*Athlete, error) {
	return s.repository.GetByID(ctx, id)
}

func (s *Service) List(
	ctx context.Context,
	params ListAthletesQuery,
) ([]Athlete, int64, error) {
	if params.Page < 1 {
		params.Page = 1
	}

	if params.PageSize < 1 {
		params.PageSize = 20
	}

	if params.PageSize > 100 {
		params.PageSize = 100
	}

	return s.repository.List(ctx, params)
}

func (s *Service) Update(
	ctx context.Context,
	id uuid.UUID,
	request UpdateAthleteRequest,
) (*Athlete, error) {
	request.Name = strings.TrimSpace(request.Name)

	if request.Name == "" {
		return nil, ErrInvalidAthlete
	}

	if _, err := parseBirthDate(request.BirthDate); err != nil {
		return nil, err
	}

	return s.repository.Update(ctx, id, request)
}

func (s *Service) Delete(
	ctx context.Context,
	id uuid.UUID,
) error {
	return s.repository.Delete(ctx, id)
}

func parseBirthDate(value *string) (*time.Time, error) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil, nil
	}

	parsed, err := time.Parse("2006-01-02", strings.TrimSpace(*value))
	if err != nil {
		return nil, errors.New("birth_date must have format YYYY-MM-DD")
	}

	return &parsed, nil
}
