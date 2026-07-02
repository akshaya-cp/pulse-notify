package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/akshaya-cp/golang_project/internal/middleware"
	"github.com/akshaya-cp/golang_project/internal/repository"
	"github.com/akshaya-cp/golang_project/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type NotificationHandler struct {
	notifications *service.NotificationService
}

func NewNotificationHandler(notifications *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{notifications: notifications}
}

type createNotificationRequest struct {
	Channel   string `json:"channel" binding:"required,oneof=email sms push"`
	Recipient string `json:"recipient" binding:"required,max=255"`
	Subject   string `json:"subject" binding:"max=255"`
	Body      string `json:"body" binding:"required,max=4000"`
}

// Create enqueues a notification for asynchronous delivery. It returns 202
// Accepted because the actual send happens later in the worker pipeline.
func (h *NotificationHandler) Create(c *gin.Context) {
	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req createNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.notifications.Enqueue(c.Request.Context(), userID, req.Channel, req.Recipient, req.Subject, req.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not enqueue notification"})
		return
	}

	c.JSON(http.StatusAccepted, resp)
}

// List returns the caller's notifications.
func (h *NotificationHandler) List(c *gin.Context) {
	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	limit := parseLimit(c.Query("limit"))
	items, err := h.notifications.ListForUser(c.Request.Context(), userID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not list notifications"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"notifications": items, "count": len(items)})
}

// Get returns a single notification the caller owns.
func (h *NotificationHandler) Get(c *gin.Context) {
	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid notification id"})
		return
	}

	resp, err := h.notifications.Get(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotificationNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "notification not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not fetch notification"})
		return
	}

	if resp.UserID != userID.String() {
		c.JSON(http.StatusNotFound, gin.H{"error": "notification not found"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// ListAll returns notifications across all users (admin only).
func (h *NotificationHandler) ListAll(c *gin.Context) {
	limit := parseLimit(c.Query("limit"))
	items, err := h.notifications.ListAll(c.Request.Context(), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not list notifications"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"notifications": items, "count": len(items)})
}

func parseLimit(raw string) int {
	if raw == "" {
		return 0
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}
	return v
}
