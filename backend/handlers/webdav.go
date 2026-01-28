package handlers

import (
	"bytes"
	"encoding/xml"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
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
	Xmlnsi   string     `xml:"xmlns:i,attr"`
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

type ResourceType struct {
	Collection *struct{} `xml:"D:collection,omitempty"`
}

type Prop struct {
	DisplayName     string       `xml:"D:displayname"`
	GetContentType  string       `xml:"D:getcontenttype,omitempty"`
	GetContentLen   int64        `xml:"D:getcontentlength,omitempty"`
	GetLastModified string       `xml:"D:getlastmodified,omitempty"`
	ResourceType    ResourceType `xml:"D:resourcetype"`
	GetEtag         string       `xml:"D:getetag,omitempty"`
}

func (h *WebDAVHandler) Propfind(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	path := c.Param("path")
	if path == "" {
		path = "/"
	}

	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Header("Content-Type", "application/xml; charset=utf-8")

	var files []models.File

	if path == "/" {
		database.DB.Where("owner_id = ? AND path = ? AND is_trashed = false", user.ID, "/").Find(&files)
	} else {
		cleanPath := strings.TrimSuffix(path, "/")
		database.DB.Where("owner_id = ? AND path = ? AND is_trashed = false", user.ID, cleanPath).Find(&files)
	}

	response := PropfindResponse{
		Xmlns:    "DAV:",
		Xmlnsi:   "DAV:",
		Response: make([]Response, 0, len(files)+1),
	}

	displayName := "root"
	if path != "/" {
		displayName = filepath.Base(strings.TrimSuffix(path, "/"))
	}
	response.Response = append(response.Response, Response{
		Href: "/webdav" + strings.TrimSuffix(path, "/") + "/",
		Propstat: Propstat{
			Prop: Prop{
				DisplayName:  displayName,
				ResourceType: ResourceType{Collection: &struct{}{}},
			},
			Status: "HTTP/1.1 200 OK",
		},
	})

	cleanPath := strings.TrimSuffix(path, "/")
	for _, file := range files {
		href := "/webdav" + cleanPath + "/" + file.Name
		prop := Prop{
			DisplayName:     file.Name,
			GetLastModified: file.UpdatedAt.Format(time.RFC1123),
			GetEtag:         "\"" + file.Checksum + "\"",
		}

		if file.IsDirectory {
			prop.ResourceType = ResourceType{Collection: &struct{}{}}
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

	c.XML(http.StatusMultiStatus, response)
}

func (h *WebDAVHandler) Get(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	path := c.Param("path")

	if path == "" {
		path = "/"
	}

	cleanPath := strings.TrimSuffix(path, "/")

	if path == "/" || strings.HasSuffix(path, "/") {
		var files []models.File

		if path == "/" {
			database.DB.Where("owner_id = ? AND path = ? AND is_trashed = false", user.ID, "/").Find(&files)
		} else {
			database.DB.Where("owner_id = ? AND path = ? AND is_trashed = false", user.ID, cleanPath).Find(&files)
		}

		response := PropfindResponse{
			Xmlns:    "DAV:",
			Xmlnsi:   "DAV:",
			Response: make([]Response, 0, len(files)+1),
		}

		response.Response = append(response.Response, Response{
			Href: "/webdav" + strings.TrimSuffix(path, "/") + "/",
			Propstat: Propstat{
				Prop: Prop{
					DisplayName:  filepath.Base(strings.TrimSuffix(path, "/")),
					ResourceType: ResourceType{Collection: &struct{}{}},
				},
				Status: "HTTP/1.1 200 OK",
			},
		})

		cleanPath := strings.TrimSuffix(path, "/")
		for _, file := range files {
			href := "/webdav" + cleanPath + "/" + file.Name
			prop := Prop{
				DisplayName:     file.Name,
				GetLastModified: file.UpdatedAt.Format(time.RFC1123),
				GetEtag:         "\"" + file.Checksum + "\"",
			}

			if file.IsDirectory {
				prop.ResourceType = ResourceType{Collection: &struct{}{}}
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
		return
	}

	pathParts := strings.Split(strings.TrimPrefix(cleanPath, "/"), "/")
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
	c.Header("Content-Length", strconv.FormatInt(file.Size, 10))
	c.Header("Content-Disposition", "inline; filename="+file.Name)
	c.Header("ETag", "\""+file.Checksum+"\"")
	c.Header("Accept-Ranges", "bytes")
	c.Header("Cache-Control", "no-cache")
	c.File(file.StoragePath)
}

func (h *WebDAVHandler) Put(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	path := c.Param("path")

	// Debug: log headers
	log.Printf("PUT request - Content-Length: %s, Transfer-Encoding: %s, Expect: %s",
		c.GetHeader("Content-Length"),
		c.GetHeader("Transfer-Encoding"),
		c.GetHeader("Expect"))

	cleanPath := strings.TrimSuffix(path, "/")
	pathParts := strings.Split(strings.TrimPrefix(cleanPath, "/"), "/")
	fileName := pathParts[len(pathParts)-1]
	parentPath := "/"
	if len(pathParts) > 1 {
		parentPath = "/" + strings.Join(pathParts[:len(pathParts)-1], "/")
	}

	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Printf("PUT error reading body: %v", err)
		c.Status(http.StatusInternalServerError)
		return
	}
	c.Request.Body.Close()

	log.Printf("PUT actual body size: %d bytes", len(bodyBytes))

	// If body is empty, just return success without creating file
	// This handles Windows WebDAV clients that send empty PUT first
	if len(bodyBytes) == 0 {
		// Check if file already exists
		var existingFile models.File
		err = database.DB.Where("owner_id = ? AND path = ? AND name = ? AND is_trashed = false", user.ID, parentPath, fileName).First(&existingFile).Error
		if err == nil {
			// File exists, return 204
			c.Status(http.StatusNoContent)
		} else {
			// File doesn't exist, return 201 but don't create
			c.Status(http.StatusCreated)
		}
		return
	}

	bodyReader := bytes.NewReader(bodyBytes)

	storagePath, size, checksum, err := h.storage.SaveFile(user.ID, bodyReader, fileName)
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

	database.DB.Create(&newFile)
	database.DB.Model(user).Update("used_space", user.UsedSpace+size)

	c.Status(http.StatusCreated)
}

func (h *WebDAVHandler) Mkcol(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	path := c.Param("path")

	cleanPath := strings.TrimSuffix(path, "/")
	pathParts := strings.Split(strings.TrimPrefix(cleanPath, "/"), "/")
	folderName := pathParts[len(pathParts)-1]
	parentPath := "/"
	if len(pathParts) > 1 {
		parentPath = "/" + strings.Join(pathParts[:len(pathParts)-1], "/")
	}

	var existingFolder models.File
	if err := database.DB.Where("owner_id = ? AND path = ? AND name = ? AND is_directory = true AND is_trashed = false", user.ID, parentPath, folderName).First(&existingFolder).Error; err == nil {
		c.Status(http.StatusConflict)
		return
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
	pathParts := strings.Split(strings.TrimPrefix(cleanPath, "/"), "/")
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
	srcParts := strings.Split(strings.TrimPrefix(cleanSrcPath, "/"), "/")
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

	if strings.Contains(destPath, "://") {
		parts := strings.SplitN(destPath, "/webdav", 2)
		if len(parts) == 2 {
			destPath = parts[1]
		}
	} else {
		destPath = strings.TrimPrefix(destPath, "/webdav")
	}

	cleanDestPath := strings.TrimSuffix(destPath, "/")
	destParts := strings.Split(strings.TrimPrefix(cleanDestPath, "/"), "/")
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
	srcParts := strings.Split(strings.TrimPrefix(cleanSrcPath, "/"), "/")
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

	if strings.Contains(destPath, "://") {
		parts := strings.SplitN(destPath, "/webdav", 2)
		if len(parts) == 2 {
			destPath = parts[1]
		}
	} else {
		destPath = strings.TrimPrefix(destPath, "/webdav")
	}

	cleanDestPath := strings.TrimSuffix(destPath, "/")
	destParts := strings.Split(strings.TrimPrefix(cleanDestPath, "/"), "/")
	destName := destParts[len(destParts)-1]
	destParentPath := "/"
	if len(destParts) > 1 {
		destParentPath = "/" + strings.Join(destParts[:len(destParts)-1], "/")
	}

	newStoragePath := filepath.Join(h.storage.GetUserStoragePath(user.ID), uuid.New().String()+filepath.Ext(file.Name))
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
	c.Header("MS-Author-Via", "DAV")
	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Status(http.StatusOK)
}

func (h *WebDAVHandler) Head(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	path := c.Param("path")

	cleanPath := strings.TrimSuffix(path, "/")
	pathParts := strings.Split(strings.TrimPrefix(cleanPath, "/"), "/")
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
	c.Header("ETag", "\""+file.Checksum+"\"")
	c.Header("Accept-Ranges", "bytes")
	c.Status(http.StatusOK)
}

func (h *WebDAVHandler) Lock(c *gin.Context) {
	path := c.Param("path")
	if path == "" {
		path = "/"
	}

	lockToken := "opaquelocktoken:" + uuid.New().String()

	lockResponse := `<?xml version="1.0" encoding="utf-8"?>
<D:prop xmlns:D="DAV:">
  <D:lockdiscovery>
    <D:activelock>
      <D:locktype><D:write/></D:locktype>
      <D:lockscope><D:exclusive/></D:lockscope>
      <D:depth>infinity</D:depth>
      <D:owner><D:href>` + c.GetHeader("X-Real-IP") + `</D:href></D:owner>
      <D:timeout>Second-3600</D:timeout>
      <D:locktoken><D:href>` + lockToken + `</D:href></D:locktoken>
      <D:lockroot><D:href>/webdav` + path + `</D:href></D:lockroot>
    </D:activelock>
  </D:lockdiscovery>
</D:prop>`

	c.Header("Lock-Token", "<"+lockToken+">")
	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.String(http.StatusOK, lockResponse)
}

func (h *WebDAVHandler) Unlock(c *gin.Context) {
	c.Status(http.StatusNoContent)
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
