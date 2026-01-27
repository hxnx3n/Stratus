package handlers

import (
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"stratus/config"
	"stratus/database"
	"stratus/middleware"
	"stratus/models"
	"stratus/services"
)

type FileHandler struct {
	config  *config.Config
	storage *services.StorageService
}

func NewFileHandler(cfg *config.Config, storage *services.StorageService) *FileHandler {
	return &FileHandler{
		config:  cfg,
		storage: storage,
	}
}

type FileListResponse struct {
	Files           []models.File `json:"files"`
	TotalCount      int64         `json:"total_count"`
	Path            string        `json:"path"`
	CurrentParentID *uuid.UUID    `json:"current_parent_id"`
}

type CreateFolderRequest struct {
	Name     string     `json:"name" binding:"required"`
	ParentID *uuid.UUID `json:"parent_id"`
}

func (h *FileHandler) List(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	path := c.DefaultQuery("path", "/")

	var files []models.File
	var currentParentID *uuid.UUID
	query := database.DB.Where("owner_id = ? AND is_trashed = false", user.ID)

	if path != "/" {
		lastSlash := strings.LastIndex(path, "/")
		var parentPath, folderName string
		if lastSlash == 0 {
			parentPath = "/"
			folderName = path[1:]
		} else {
			parentPath = path[:lastSlash]
			folderName = path[lastSlash+1:]
		}

		var parent models.File
		if err := database.DB.Where("owner_id = ? AND path = ? AND name = ? AND is_directory = true", user.ID, parentPath, folderName).First(&parent).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Directory not found"})
			return
		}
		currentParentID = &parent.ID
		query = query.Where("parent_id = ?", parent.ID)
	} else {
		query = query.Where("parent_id IS NULL")
	}

	query.Order("is_directory DESC, name ASC").Find(&files)

	var totalCount int64
	database.DB.Model(&models.File{}).Where("owner_id = ? AND is_trashed = false", user.ID).Count(&totalCount)

	c.JSON(http.StatusOK, gin.H{
		"files":             files,
		"total_count":       totalCount,
		"path":              path,
		"current_parent_id": currentParentID,
	})
}

func (h *FileHandler) Get(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	fileID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file ID"})
		return
	}

	var file models.File
	if err := database.DB.Where("id = ? AND owner_id = ?", fileID, user.ID).First(&file).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	c.JSON(http.StatusOK, file)
}

func (h *FileHandler) GetContents(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	folderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid folder ID"})
		return
	}

	var files []models.File
	if err := database.DB.Where("owner_id = ? AND parent_id = ? AND is_trashed = false", user.ID, folderID).
		Order("is_directory DESC, name ASC").
		Find(&files).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch contents"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"files": files})
}

func (h *FileHandler) Upload(c *gin.Context) {
	user := middleware.GetCurrentUser(c)

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file provided"})
		return
	}
	defer file.Close()

	if header.Size > h.config.MaxUploadSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File too large"})
		return
	}

	if !user.HasEnoughSpace(header.Size) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Storage quota exceeded"})
		return
	}

	var parentID *uuid.UUID
	var parentPath string = "/"

	parentIDStr := c.PostForm("parent_id")
	if parentIDStr != "" {
		if parsedID, err := uuid.Parse(parentIDStr); err == nil {
			var parent models.File
			if err := database.DB.Where("id = ? AND owner_id = ? AND is_directory = true", parsedID, user.ID).First(&parent).Error; err == nil {
				parentID = &parent.ID
				if parent.Path == "/" {
					parentPath = "/" + parent.Name
				} else {
					parentPath = parent.Path + "/" + parent.Name
				}
			}
		}
	}

	storagePath, size, checksum, err := h.storage.SaveFile(user.ID, file, header.Filename)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}

	var existingFile models.File
	if parentID != nil {
		err = database.DB.Where("owner_id = ? AND parent_id = ? AND name = ? AND is_trashed = false", user.ID, parentID, header.Filename).First(&existingFile).Error
	} else {
		err = database.DB.Where("owner_id = ? AND parent_id IS NULL AND name = ? AND is_trashed = false", user.ID, header.Filename).First(&existingFile).Error
	}

	if err == nil {
		h.storage.CreateFileVersion(&existingFile)

		existingFile.Size = size
		existingFile.StoragePath = storagePath
		existingFile.Checksum = checksum
		existingFile.Version++
		database.DB.Save(&existingFile)

		c.JSON(http.StatusOK, existingFile)
		return
	}

	newFile := models.File{
		Name:        header.Filename,
		Path:        parentPath,
		StoragePath: storagePath,
		MimeType:    h.storage.GetMimeType(header.Filename),
		Size:        size,
		IsDirectory: false,
		ParentID:    parentID,
		OwnerID:     user.ID,
		Checksum:    checksum,
	}

	if err := database.DB.Create(&newFile).Error; err != nil {
		h.storage.DeleteFile(storagePath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create file record"})
		return
	}

	database.DB.Model(user).Update("used_space", user.UsedSpace+size)

	activity := models.Activity{
		UserID:   user.ID,
		Type:     models.ActivityFileCreated,
		FileID:   &newFile.ID,
		FileName: newFile.Name,
	}
	database.DB.Create(&activity)

	c.JSON(http.StatusCreated, newFile)
}

func (h *FileHandler) Download(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	fileID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file ID"})
		return
	}

	var file models.File
	if err := database.DB.Where("id = ? AND owner_id = ? AND is_directory = false", fileID, user.ID).First(&file).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	activity := models.Activity{
		UserID:   user.ID,
		Type:     models.ActivityFileDownloaded,
		FileID:   &file.ID,
		FileName: file.Name,
	}
	database.DB.Create(&activity)

	c.Header("Content-Disposition", "attachment; filename="+file.Name)
	c.Header("Content-Type", file.MimeType)
	c.File(file.StoragePath)
}

func (h *FileHandler) CreateFolder(c *gin.Context) {
	user := middleware.GetCurrentUser(c)

	var req CreateFolderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if strings.ContainsAny(req.Name, "/\\:*?\"<>|") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid folder name"})
		return
	}

	var parentPath string = "/"
	if req.ParentID != nil {
		var parent models.File
		if err := database.DB.Where("id = ? AND owner_id = ? AND is_directory = true", req.ParentID, user.ID).First(&parent).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Parent folder not found"})
			return
		}
		if parent.Path == "/" {
			parentPath = "/" + parent.Name
		} else {
			parentPath = parent.Path + "/" + parent.Name
		}
	}

	var existingFolder models.File
	var folderErr error
	if req.ParentID != nil {
		folderErr = database.DB.Where("owner_id = ? AND parent_id = ? AND name = ? AND is_directory = true AND is_trashed = false", user.ID, req.ParentID, req.Name).First(&existingFolder).Error
	} else {
		folderErr = database.DB.Where("owner_id = ? AND parent_id IS NULL AND name = ? AND is_directory = true AND is_trashed = false", user.ID, req.Name).First(&existingFolder).Error
	}

	if folderErr == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Folder already exists"})
		return
	}

	folder := models.File{
		Name:        req.Name,
		Path:        parentPath,
		IsDirectory: true,
		ParentID:    req.ParentID,
		OwnerID:     user.ID,
		StoragePath: filepath.Join(user.ID.String(), uuid.New().String()),
	}

	if err := database.DB.Create(&folder).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create folder"})
		return
	}

	activity := models.Activity{
		UserID:   user.ID,
		Type:     models.ActivityFolderCreated,
		FileID:   &folder.ID,
		FileName: folder.Name,
	}
	database.DB.Create(&activity)

	c.JSON(http.StatusCreated, folder)
}

func (h *FileHandler) Rename(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	fileID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file ID"})
		return
	}

	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var file models.File
	if err := database.DB.Where("id = ? AND owner_id = ?", fileID, user.ID).First(&file).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	file.Name = req.Name
	database.DB.Save(&file)

	c.JSON(http.StatusOK, file)
}

func (h *FileHandler) Move(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	fileID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file ID"})
		return
	}

	var req struct {
		DestinationID *uuid.UUID `json:"destination_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var file models.File
	if err := database.DB.Where("id = ? AND owner_id = ?", fileID, user.ID).First(&file).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	var newPath string = "/"
	if req.DestinationID != nil {
		var dest models.File
		if err := database.DB.Where("id = ? AND owner_id = ? AND is_directory = true", req.DestinationID, user.ID).First(&dest).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Destination folder not found"})
			return
		}
		newPath = filepath.Join(dest.Path, dest.Name)
	}

	file.ParentID = req.DestinationID
	file.Path = newPath
	database.DB.Save(&file)

	activity := models.Activity{
		UserID:   user.ID,
		Type:     models.ActivityFileMoved,
		FileID:   &file.ID,
		FileName: file.Name,
		Details:  "Moved to " + newPath,
	}
	database.DB.Create(&activity)

	c.JSON(http.StatusOK, file)
}

func (h *FileHandler) Copy(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	fileID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file ID"})
		return
	}

	var req struct {
		DestinationID *uuid.UUID `json:"destination_id"`
		NewName       string     `json:"new_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var file models.File
	if err := database.DB.Where("id = ? AND owner_id = ? AND is_directory = false", fileID, user.ID).First(&file).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	if !user.HasEnoughSpace(file.Size) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Storage quota exceeded"})
		return
	}

	var newPath string = "/"
	if req.DestinationID != nil {
		var dest models.File
		if err := database.DB.Where("id = ? AND owner_id = ? AND is_directory = true", req.DestinationID, user.ID).First(&dest).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Destination folder not found"})
			return
		}
		newPath = filepath.Join(dest.Path, dest.Name)
	}

	newStoragePath := filepath.Join(h.storage.GetUserStoragePath(user.ID), uuid.New().String()+filepath.Ext(file.Name))
	if err := h.storage.CopyFile(file.StoragePath, newStoragePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to copy file"})
		return
	}

	newName := req.NewName
	if newName == "" {
		newName = file.Name
	}

	newFile := models.File{
		Name:        newName,
		Path:        newPath,
		StoragePath: newStoragePath,
		MimeType:    file.MimeType,
		Size:        file.Size,
		IsDirectory: false,
		ParentID:    req.DestinationID,
		OwnerID:     user.ID,
		Checksum:    file.Checksum,
	}

	if err := database.DB.Create(&newFile).Error; err != nil {
		h.storage.DeleteFile(newStoragePath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create file record"})
		return
	}

	database.DB.Model(user).Update("used_space", user.UsedSpace+file.Size)

	c.JSON(http.StatusCreated, newFile)
}

func (h *FileHandler) Trash(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	fileID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file ID"})
		return
	}

	var file models.File
	if err := database.DB.Where("id = ? AND owner_id = ?", fileID, user.ID).First(&file).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	now := time.Now()
	file.IsTrashed = true
	file.TrashedAt = &now
	database.DB.Save(&file)

	c.JSON(http.StatusOK, gin.H{"message": "File moved to trash"})
}

func (h *FileHandler) Restore(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	fileID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file ID"})
		return
	}

	var file models.File
	if err := database.DB.Where("id = ? AND owner_id = ? AND is_trashed = true", fileID, user.ID).First(&file).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found in trash"})
		return
	}

	file.IsTrashed = false
	file.TrashedAt = nil
	database.DB.Save(&file)

	c.JSON(http.StatusOK, file)
}

func (h *FileHandler) Delete(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	fileID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file ID"})
		return
	}

	var file models.File
	if err := database.DB.Where("id = ? AND owner_id = ?", fileID, user.ID).First(&file).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	if !file.IsDirectory {
		h.storage.DeleteFile(file.StoragePath)

		database.DB.Model(user).Update("used_space", user.UsedSpace-file.Size)
	}

	database.DB.Where("file_id = ?", file.ID).Delete(&models.FileVersion{})

	database.DB.Delete(&file)

	activity := models.Activity{
		UserID:   user.ID,
		Type:     models.ActivityFileDeleted,
		FileName: file.Name,
	}
	database.DB.Create(&activity)

	c.JSON(http.StatusOK, gin.H{"message": "File deleted permanently"})
}

func (h *FileHandler) ListTrash(c *gin.Context) {
	user := middleware.GetCurrentUser(c)

	var files []models.File
	database.DB.Where("owner_id = ? AND is_trashed = true", user.ID).
		Order("trashed_at DESC").
		Find(&files)

	c.JSON(http.StatusOK, gin.H{"files": files})
}

func (h *FileHandler) EmptyTrash(c *gin.Context) {
	user := middleware.GetCurrentUser(c)

	var files []models.File
	database.DB.Where("owner_id = ? AND is_trashed = true", user.ID).Find(&files)

	var freedSpace int64
	for _, file := range files {
		if !file.IsDirectory {
			h.storage.DeleteFile(file.StoragePath)
			freedSpace += file.Size
		}
		database.DB.Where("file_id = ?", file.ID).Delete(&models.FileVersion{})
		database.DB.Delete(&file)
	}

	database.DB.Model(user).Update("used_space", user.UsedSpace-freedSpace)

	c.JSON(http.StatusOK, gin.H{"message": "Trash emptied"})
}

func (h *FileHandler) Search(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Search query required"})
		return
	}

	var files []models.File
	database.DB.Where("owner_id = ? AND is_trashed = false AND name ILIKE ?", user.ID, "%"+query+"%").
		Order("is_directory DESC, name ASC").
		Find(&files)

	c.JSON(http.StatusOK, files)
}

func (h *FileHandler) StorageStats(c *gin.Context) {
	user := middleware.GetCurrentUser(c)

	usedSpace, _ := h.storage.GetStorageUsage(user.ID)

	var fileCount int64
	database.DB.Model(&models.File{}).Where("owner_id = ? AND is_directory = false AND is_trashed = false", user.ID).Count(&fileCount)

	var folderCount int64
	database.DB.Model(&models.File{}).Where("owner_id = ? AND is_directory = true AND is_trashed = false", user.ID).Count(&folderCount)

	c.JSON(http.StatusOK, gin.H{
		"used_space":   usedSpace,
		"quota":        user.Quota,
		"file_count":   fileCount,
		"folder_count": folderCount,
		"percentage":   float64(usedSpace) / float64(user.Quota) * 100,
	})
}
