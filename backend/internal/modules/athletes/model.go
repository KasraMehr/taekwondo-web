package athletes

import (
	"time"

	"github.com/google/uuid"
)

// Athlete represents the athlete stored in PostgreSQL.
type Athlete struct {
	ID uuid.UUID `json:"id"`

	Name string `json:"name"`

	NationalCode *string    `json:"national_code,omitempty"`
	BirthDate    *time.Time `json:"birth_date,omitempty"`
	Gender       *string    `json:"gender,omitempty"`
	Phone        *string    `json:"phone,omitempty"`
	Email        *string    `json:"email,omitempty" binding:"omitempty,email,max=255"`

	IsActive bool `json:"is_active"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateAthleteRequest is the body of POST /api/v1/athletes.
type CreateAthleteRequest struct {
	Name string `json:"name" binding:"required,max=100"`

	NationalCode *string `json:"national_code,omitempty" binding:"omitempty,max=20"`
	BirthDate    *string `json:"birth_date,omitempty" binding:"omitempty,datetime=2006-01-02"`
	Gender       *string `json:"gender,omitempty" binding:"omitempty,oneof=male female"`
	Phone        *string `json:"phone,omitempty" binding:"omitempty,max=30"`
	Email        *string `json:"email,omitempty" binding:"omitempty,email,max=255"`
}

// UpdateAthleteRequest is the body of PUT /api/v1/athletes/:id.
type UpdateAthleteRequest struct {
	Name string `json:"name" binding:"required,max=100"`

	NationalCode *string `json:"national_code,omitempty" binding:"omitempty,max=20"`
	BirthDate    *string `json:"birth_date,omitempty" binding:"omitempty,datetime=2006-01-02"`
	Gender       *string `json:"gender,omitempty" binding:"omitempty,oneof=male female"`
	Phone        *string `json:"phone,omitempty" binding:"omitempty,max=30"`
	Email        *string `json:"email,omitempty" binding:"omitempty,email,max=255"`

	IsActive bool `json:"is_active"`
}

// ListAthletesQuery contains GET /api/v1/athletes query parameters.
type ListAthletesQuery struct {
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
	Search   string `form:"search"`
	IsActive *bool  `form:"is_active"`
}

// ListAthletesResponse is the paginated API response.
type ListAthletesResponse struct {
	Items      []Athlete `json:"items"`
	Page       int       `json:"page"`
	PageSize   int       `json:"page_size"`
	Total      int64     `json:"total"`
	TotalPages int       `json:"total_pages"`
}
