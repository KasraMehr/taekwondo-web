package athletes

import (
	"errors"
	"math"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) Create(c *gin.Context) {
	var request CreateAthleteRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	athlete, err := h.service.Create(c.Request.Context(), request)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, athlete)
}

func (h *Handler) GetByID(c *gin.Context) {
	id, err := parseUUIDParam(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_id",
			"message": "id must be a valid UUID",
		})
		return
	}

	athlete, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, athlete)
}

func (h *Handler) List(c *gin.Context) {
	var query ListAthletesQuery

	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_query",
			"message": err.Error(),
		})
		return
	}

	if query.Page < 1 {
		query.Page = 1
	}

	if query.PageSize < 1 {
		query.PageSize = 20
	}

	if query.PageSize > 100 {
		query.PageSize = 100
	}

	items, total, err := h.service.List(c.Request.Context(), query)
	if err != nil {
		h.handleError(c, err)
		return
	}

	totalPages := 0
	if total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(query.PageSize)))
	}

	response := ListAthletesResponse{
		Items:      items,
		Page:       query.Page,
		PageSize:   query.PageSize,
		Total:      total,
		TotalPages: totalPages,
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) Update(c *gin.Context) {
	id, err := parseUUIDParam(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_id",
			"message": "id must be a valid UUID",
		})
		return
	}

	var request UpdateAthleteRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	athlete, err := h.service.Update(
		c.Request.Context(),
		id,
		request,
	)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, athlete)
}

func (h *Handler) Delete(c *gin.Context) {
	id, err := parseUUIDParam(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_id",
			"message": "id must be a valid UUID",
		})
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		h.handleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func parseUUIDParam(c *gin.Context) (uuid.UUID, error) {
	value := c.Param("id")
	return uuid.Parse(value)
}

func (h *Handler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": "athlete not found",
		})

	case errors.Is(err, ErrInvalidAthlete):
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_athlete",
			"message": err.Error(),
		})

	default:
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_server_error",
			"message": "an internal server error occurred",
		})
	}
}
