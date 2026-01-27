package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"stratus/config"
	"stratus/database"
	"stratus/middleware"
	"stratus/models"
)

type AdminHandler struct {
	config *config.Config
}

func NewAdminHandler(cfg *config.Config) *AdminHandler {
	return &AdminHandler{config: cfg}
}

func (h *AdminHandler) ListUsers(c *gin.Context) {
	var users []models.User
	database.DB.Order("created_at DESC").Find(&users)
	c.JSON(http.StatusOK, users)
}

func (h *AdminHandler) GetUser(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}

func (h *AdminHandler) UpdateUser(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	var req struct {
		DisplayName string `json:"display_name"`
		Email       string `json:"email"`
		Quota       *int64 `json:"quota"`
		IsActive    *bool  `json:"is_active"`
		IsAdmin     *bool  `json:"is_admin"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := make(map[string]interface{})
	if req.DisplayName != "" {
		updates["display_name"] = req.DisplayName
	}
	if req.Email != "" {
		updates["email"] = req.Email
	}
	if req.Quota != nil {
		updates["quota"] = *req.Quota
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}
	if req.IsAdmin != nil {
		currentUser := middleware.GetCurrentUser(c)
		if currentUser.ID == user.ID && !*req.IsAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "Cannot remove your own admin status"})
			return
		}
		updates["is_admin"] = *req.IsAdmin
	}

	database.DB.Model(&user).Updates(updates)
	database.DB.First(&user, userID)

	c.JSON(http.StatusOK, user)
}

func (h *AdminHandler) DeleteUser(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	currentUser := middleware.GetCurrentUser(c)
	if currentUser.ID == userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot delete yourself"})
		return
	}

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	database.DB.Where("owner_id = ?", userID).Delete(&models.File{})
	database.DB.Where("user_id = ?", userID).Delete(&models.Activity{})
	database.DB.Delete(&user)

	c.JSON(http.StatusOK, gin.H{"message": "User deleted"})
}

func (h *AdminHandler) SystemStats(c *gin.Context) {
	var userCount int64
	database.DB.Model(&models.User{}).Count(&userCount)

	var fileCount int64
	database.DB.Model(&models.File{}).Where("is_directory = false").Count(&fileCount)

	var folderCount int64
	database.DB.Model(&models.File{}).Where("is_directory = true").Count(&folderCount)

	var totalSize int64
	database.DB.Model(&models.File{}).
		Where("is_directory = false").
		Select("COALESCE(SUM(size), 0)").
		Scan(&totalSize)

	var shareCount int64

	c.JSON(http.StatusOK, gin.H{
		"users":      userCount,
		"files":      fileCount,
		"folders":    folderCount,
		"total_size": totalSize,
		"shares":     shareCount,
	})
}

func (h *AdminHandler) ListActivities(c *gin.Context) {
	var activities []models.Activity
	database.DB.Preload("User").
		Order("created_at DESC").
		Limit(100).
		Find(&activities)

	c.JSON(http.StatusOK, activities)
}
