package handlers

import (
	"encoding/xml"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"stratus/config"
	"stratus/database"
	"stratus/middleware"
	"stratus/models"
	"stratus/services"
)

type WebDAVHandler struct {
	config  *config.Config
	storage *services.StorageService
}

func NewWebDAVHandler(cfg *config.Config, storage *services.StorageService) *WebDAVHandler {
	return &WebDAVHandler{
		config:  cfg,
		storage: storage,
	}
}

type PropfindResponse struct {
	XMLName  xml.Name   `xml:"D:multistatus"`
	Xmlns    string     `xml:"xmlns:D,attr"`
	Response []Response `xml:"D:response"`
}

type Response struct {
	Href     string   `xml:"D:href"`
	Propstat Propstat `xml:"D:propstat"`
}

type Propstat struct {
	Prop   Prop   `xml:"D:prop"`
	Status string `xml:"D:status"`
}

type Prop struct {
	DisplayName     string `xml:"D:displayname"`
	GetContentType  string `xml:"D:getcontenttype,omitempty"`
	GetContentLen   int64  `xml:"D:getcontentlength,omitempty"`
	GetLastModified string `xml:"D:getlastmodified,omitempty"`
	ResourceType    string `xml:"D:resourcetype,omitempty"`
	GetEtag         string `xml:"D:getetag,omitempty"`
}

func (h *WebDAVHandler) Propfind(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	path := c.Param("path")
	if path == "" {
		path = "/"
	}

	var files []models.File

	if path == "/" {
		database.DB.Where("owner_id = ? AND parent_id IS NULL AND is_trashed = false", user.ID).Find(&files)
	} else {
		var folder models.File
		cleanPath := strings.TrimSuffix(path, "/")
		pathParts := strings.Split(cleanPath, "/")
		fileName := pathParts[len(pathParts)-1]
		parentPath := "/" + strings.Join(pathParts[:len(pathParts)-1], "/")

		if err := database.DB.Where("owner_id = ? AND path = ? AND name = ? AND is_trashed = false", user.ID, parentPath, fileName).First(&folder).Error; err != nil {
			c.Status(http.StatusNotFound)
			return
		}

		database.DB.Where("owner_id = ? AND parent_id = ? AND is_trashed = false", user.ID, folder.ID).Find(&files)
	}

	response := PropfindResponse{
		Xmlns:    "DAV:",
		Response: make([]Response, 0, len(files)+1),
	}

	response.Response = append(response.Response, Response{
		Href: "/webdav" + path,
		Propstat: Propstat{
			Prop: Prop{
				DisplayName:  filepath.Base(path),
				ResourceType: "<D:collection/>",
			},
			Status: "HTTP/1.1 200 OK",
		},
	})

	for _, file := range files {
		href := filepath.Join("/webdav", path, file.Name)
		prop := Prop{
			DisplayName:     file.Name,
			GetLastModified: file.UpdatedAt.Format(time.RFC1123),
			GetEtag:         file.Checksum,
		}

		if file.IsDirectory {
			prop.ResourceType = "<D:collection/>"
			href += "/"
		} else {
			prop.GetContentType = file.MimeType
			prop.GetContentLen = file.Size
		}

		response.Response = append(response.Response, Response{
			Href: href,
			Propstat: Propstat{
				Prop:   prop,
				Status: "HTTP/1.1 200 OK",
			},
		})
	}

	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.XML(http.StatusMultiStatus, response)
}

func (h *WebDAVHandler) Get(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	path := c.Param("path")

	cleanPath := strings.TrimSuffix(path, "/")
	pathParts := strings.Split(cleanPath, "/")
	fileName := pathParts[len(pathParts)-1]
	parentPath := "/"
	if len(pathParts) > 1 {
		parentPath = "/" + strings.Join(pathParts[:len(pathParts)-1], "/")
	}

	var file models.File
	if err := database.DB.Where("owner_id = ? AND path = ? AND name = ? AND is_directory = false AND is_trashed = false", user.ID, parentPath, fileName).First(&file).Error; err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	c.Header("Content-Type", file.MimeType)
	c.Header("Content-Disposition", "attachment; filename="+file.Name)
	c.Header("ETag", file.Checksum)
	c.File(file.StoragePath)
}

func (h *WebDAVHandler) Put(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	path := c.Param("path")

	cleanPath := strings.TrimSuffix(path, "/")
	pathParts := strings.Split(cleanPath, "/")
	fileName := pathParts[len(pathParts)-1]
	parentPath := "/"
	if len(pathParts) > 1 {
		parentPath = "/" + strings.Join(pathParts[:len(pathParts)-1], "/")
	}

	body := c.Request.Body
	defer body.Close()

	storagePath, size, checksum, err := h.storage.SaveFile(user.ID, body, fileName)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	var existingFile models.File
	err = database.DB.Where("owner_id = ? AND path = ? AND name = ? AND is_trashed = false", user.ID, parentPath, fileName).First(&existingFile).Error

	if err == nil {
		h.storage.CreateFileVersion(&existingFile)
		existingFile.Size = size
		existingFile.StoragePath = storagePath
		existingFile.Checksum = checksum
		existingFile.Version++
		database.DB.Save(&existingFile)
		c.Status(http.StatusNoContent)
		return
	}

	var parentID *string
	if parentPath != "/" {
		var parent models.File
		parentParts := strings.Split(strings.TrimPrefix(parentPath, "/"), "/")
		parentName := parentParts[len(parentParts)-1]
		grandparentPath := "/"
		if len(parentParts) > 1 {
			grandparentPath = "/" + strings.Join(parentParts[:len(parentParts)-1], "/")
		}

		if err := database.DB.Where("owner_id = ? AND path = ? AND name = ? AND is_directory = true", user.ID, grandparentPath, parentName).First(&parent).Error; err == nil {
			id := parent.ID.String()
			parentID = &id
		}
	}

	newFile := models.File{
		Name:        fileName,
		Path:        parentPath,
		StoragePath: storagePath,
		MimeType:    h.storage.GetMimeType(fileName),
		Size:        size,
		IsDirectory: false,
		OwnerID:     user.ID,
		Checksum:    checksum,
	}

	if parentID != nil {
	}

	database.DB.Create(&newFile)
	database.DB.Model(user).Update("used_space", user.UsedSpace+size)

	c.Status(http.StatusCreated)
}

func (h *WebDAVHandler) Mkcol(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	path := c.Param("path")

	cleanPath := strings.TrimSuffix(path, "/")
	pathParts := strings.Split(cleanPath, "/")
	folderName := pathParts[len(pathParts)-1]
	parentPath := "/"
	if len(pathParts) > 1 {
		parentPath = "/" + strings.Join(pathParts[:len(pathParts)-1], "/")
	}

	folder := models.File{
		Name:        folderName,
		Path:        parentPath,
		IsDirectory: true,
		OwnerID:     user.ID,
		StoragePath: user.ID.String(),
	}

	if err := database.DB.Create(&folder).Error; err != nil {
		c.Status(http.StatusConflict)
		return
	}

	c.Status(http.StatusCreated)
}

func (h *WebDAVHandler) Delete(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	path := c.Param("path")

	cleanPath := strings.TrimSuffix(path, "/")
	pathParts := strings.Split(cleanPath, "/")
	fileName := pathParts[len(pathParts)-1]
	parentPath := "/"
	if len(pathParts) > 1 {
		parentPath = "/" + strings.Join(pathParts[:len(pathParts)-1], "/")
	}

	var file models.File
	if err := database.DB.Where("owner_id = ? AND path = ? AND name = ? AND is_trashed = false", user.ID, parentPath, fileName).First(&file).Error; err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	if !file.IsDirectory {
		h.storage.DeleteFile(file.StoragePath)
		database.DB.Model(user).Update("used_space", user.UsedSpace-file.Size)
	}

	database.DB.Delete(&file)
	c.Status(http.StatusNoContent)
}

func (h *WebDAVHandler) Move(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	srcPath := c.Param("path")
	destPath := c.GetHeader("Destination")

	cleanSrcPath := strings.TrimSuffix(srcPath, "/")
	srcParts := strings.Split(cleanSrcPath, "/")
	srcName := srcParts[len(srcParts)-1]
	srcParentPath := "/"
	if len(srcParts) > 1 {
		srcParentPath = "/" + strings.Join(srcParts[:len(srcParts)-1], "/")
	}

	var file models.File
	if err := database.DB.Where("owner_id = ? AND path = ? AND name = ? AND is_trashed = false", user.ID, srcParentPath, srcName).First(&file).Error; err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	destPath = strings.TrimPrefix(destPath, c.Request.Host)
	destPath = strings.TrimPrefix(destPath, "/webdav")
	cleanDestPath := strings.TrimSuffix(destPath, "/")
	destParts := strings.Split(cleanDestPath, "/")
	destName := destParts[len(destParts)-1]
	destParentPath := "/"
	if len(destParts) > 1 {
		destParentPath = "/" + strings.Join(destParts[:len(destParts)-1], "/")
	}

	file.Name = destName
	file.Path = destParentPath
	database.DB.Save(&file)

	c.Status(http.StatusCreated)
}

func (h *WebDAVHandler) Copy(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	srcPath := c.Param("path")
	destPath := c.GetHeader("Destination")

	cleanSrcPath := strings.TrimSuffix(srcPath, "/")
	srcParts := strings.Split(cleanSrcPath, "/")
	srcName := srcParts[len(srcParts)-1]
	srcParentPath := "/"
	if len(srcParts) > 1 {
		srcParentPath = "/" + strings.Join(srcParts[:len(srcParts)-1], "/")
	}

	var file models.File
	if err := database.DB.Where("owner_id = ? AND path = ? AND name = ? AND is_directory = false AND is_trashed = false", user.ID, srcParentPath, srcName).First(&file).Error; err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	destPath = strings.TrimPrefix(destPath, c.Request.Host)
	destPath = strings.TrimPrefix(destPath, "/webdav")
	cleanDestPath := strings.TrimSuffix(destPath, "/")
	destParts := strings.Split(cleanDestPath, "/")
	destName := destParts[len(destParts)-1]
	destParentPath := "/"
	if len(destParts) > 1 {
		destParentPath = "/" + strings.Join(destParts[:len(destParts)-1], "/")
	}

	newStoragePath := filepath.Join(h.storage.GetUserStoragePath(user.ID), filepath.Base(file.StoragePath))
	if err := h.storage.CopyFile(file.StoragePath, newStoragePath); err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	newFile := models.File{
		Name:        destName,
		Path:        destParentPath,
		StoragePath: newStoragePath,
		MimeType:    file.MimeType,
		Size:        file.Size,
		IsDirectory: false,
		OwnerID:     user.ID,
		Checksum:    file.Checksum,
	}

	database.DB.Create(&newFile)
	database.DB.Model(user).Update("used_space", user.UsedSpace+file.Size)

	c.Status(http.StatusCreated)
}

func (h *WebDAVHandler) Options(c *gin.Context) {
	c.Header("Allow", "OPTIONS, GET, HEAD, PUT, DELETE, MKCOL, COPY, MOVE, PROPFIND")
	c.Header("DAV", "1, 2")
	c.Status(http.StatusOK)
}

func (h *WebDAVHandler) Head(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	path := c.Param("path")

	cleanPath := strings.TrimSuffix(path, "/")
	pathParts := strings.Split(cleanPath, "/")
	fileName := pathParts[len(pathParts)-1]
	parentPath := "/"
	if len(pathParts) > 1 {
		parentPath = "/" + strings.Join(pathParts[:len(pathParts)-1], "/")
	}

	var file models.File
	if err := database.DB.Where("owner_id = ? AND path = ? AND name = ? AND is_trashed = false", user.ID, parentPath, fileName).First(&file).Error; err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	c.Header("Content-Type", file.MimeType)
	c.Header("Content-Length", strconv.FormatInt(file.Size, 10))
	c.Header("ETag", file.Checksum)
	c.Status(http.StatusOK)
}

type PropfindRequest struct {
	XMLName xml.Name `xml:"propfind"`
	Prop    struct{} `xml:"prop"`
}

func (h *WebDAVHandler) parsePropfind(body io.Reader) (*PropfindRequest, error) {
	var req PropfindRequest
	decoder := xml.NewDecoder(body)
	if err := decoder.Decode(&req); err != nil {
		return nil, err
	}
	return &req, nil
}
