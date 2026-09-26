package athletes

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("athlete not found")

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{
		db: db,
	}
}

func (r *Repository) Create(
	ctx context.Context,
	athlete *Athlete,
) error {
	const query = `
		INSERT INTO athletes (
			id,
			name,
			national_code,
			birth_date,
			gender,
			phone,
			email,
			is_active
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at, updated_at
	`

	err := r.db.QueryRowContext(
		ctx,
		query,
		athlete.ID,
		athlete.Name,
		athlete.NationalCode,
		athlete.BirthDate,
		athlete.Gender,
		athlete.Phone,
		athlete.Email,
		athlete.IsActive,
	).Scan(
		&athlete.CreatedAt,
		&athlete.UpdatedAt,
	)

	return err
}

func (r *Repository) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (*Athlete, error) {
	const query = `
		SELECT
			id,
			name,
			national_code,
			birth_date,
			gender,
			phone,
			email,
			is_active,
			created_at,
			updated_at
		FROM athletes
		WHERE id = $1
	`

	athlete := &Athlete{}

	err := r.db.QueryRowContext(
		ctx,
		query,
		id,
	).Scan(
		&athlete.ID,
		&athlete.Name,
		&athlete.NationalCode,
		&athlete.BirthDate,
		&athlete.Gender,
		&athlete.Phone,
		&athlete.Email,
		&athlete.IsActive,
		&athlete.CreatedAt,
		&athlete.UpdatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}

	if err != nil {
		return nil, err
	}

	return athlete, nil
}

func (r *Repository) List(
	ctx context.Context,
	params ListAthletesQuery,
) ([]Athlete, int64, error) {
	page := params.Page
	if page < 1 {
		page = 1
	}

	pageSize := params.PageSize
	if pageSize < 1 {
		pageSize = 20
	}

	if pageSize > 100 {
		pageSize = 100
	}

	offset := (page - 1) * pageSize

	conditions := []string{"1 = 1"}
	args := make([]any, 0)
	argPosition := 1

	if search := strings.TrimSpace(params.Search); search != "" {
		conditions = append(
			conditions,
			fmt.Sprintf("name ILIKE $%d", argPosition),
		)
		args = append(args, "%"+search+"%")
		argPosition++
	}

	if params.IsActive != nil {
		conditions = append(
			conditions,
			fmt.Sprintf("is_active = $%d", argPosition),
		)
		args = append(args, *params.IsActive)
		argPosition++
	}

	whereClause := strings.Join(conditions, " AND ")

	countQuery := `
		SELECT COUNT(*)
		FROM athletes
		WHERE ` + whereClause

	var total int64

	if err := r.db.QueryRowContext(
		ctx,
		countQuery,
		args...,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	listQuery := fmt.Sprintf(`
		SELECT
			id,
			name,
			national_code,
			birth_date,
			gender,
			phone,
			email,
			is_active,
			created_at,
			updated_at
		FROM athletes
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argPosition, argPosition+1)

	listArgs := append(
		append([]any{}, args...),
		pageSize,
		offset,
	)

	rows, err := r.db.QueryContext(
		ctx,
		listQuery,
		listArgs...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	athletes := make([]Athlete, 0, pageSize)

	for rows.Next() {
		var athlete Athlete

		if err := rows.Scan(
			&athlete.ID,
			&athlete.Name,
			&athlete.NationalCode,
			&athlete.BirthDate,
			&athlete.Gender,
			&athlete.Phone,
			&athlete.Email,
			&athlete.IsActive,
			&athlete.CreatedAt,
			&athlete.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}

		athletes = append(athletes, athlete)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return athletes, total, nil
}

func (r *Repository) Update(
	ctx context.Context,
	id uuid.UUID,
	request UpdateAthleteRequest,
) (*Athlete, error) {
	const query = `
		UPDATE athletes
		SET
			name = $2,
			national_code = $3,
			birth_date = $4,
			gender = $5,
			phone = $6,
			email = $7,
			is_active = $8,
			updated_at = NOW()
		WHERE id = $1
		RETURNING
			id,
			name,
			national_code,
			birth_date,
			gender,
			phone,
			email,
			is_active,
			created_at,
			updated_at
	`

	birthDate, err := parseBirthDate(request.BirthDate)
	if err != nil {
		return nil, err
	}

	athlete := &Athlete{}

	err = r.db.QueryRowContext(
		ctx,
		query,
		id,
		request.Name,
		request.NationalCode,
		birthDate,
		request.Gender,
		request.Phone,
		request.Email,
		request.IsActive,
	).Scan(
		&athlete.ID,
		&athlete.Name,
		&athlete.NationalCode,
		&athlete.BirthDate,
		&athlete.Gender,
		&athlete.Phone,
		&athlete.Email,
		&athlete.IsActive,
		&athlete.CreatedAt,
		&athlete.UpdatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}

	if err != nil {
		return nil, err
	}

	return athlete, nil
}

func (r *Repository) Delete(
	ctx context.Context,
	id uuid.UUID,
) error {
	const query = `
		DELETE FROM athletes
		WHERE id = $1
	`

	result, err := r.db.ExecContext(
		ctx,
		query,
		id,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}
