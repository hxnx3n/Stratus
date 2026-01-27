package services

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"stratus/config"
	"stratus/database"
	"stratus/models"
)

type StorageService struct {
	config *config.Config
}

func NewStorageService(cfg *config.Config) *StorageService {
	return &StorageService{config: cfg}
}

func (s *StorageService) InitStorage() error {
	return os.MkdirAll(s.config.StoragePath, 0755)
}

func (s *StorageService) GetUserStoragePath(userID uuid.UUID) string {
	return filepath.Join(s.config.StoragePath, userID.String())
}

func (s *StorageService) EnsureUserStorage(userID uuid.UUID) error {
	path := s.GetUserStoragePath(userID)
	return os.MkdirAll(path, 0755)
}

func (s *StorageService) SaveFile(userID uuid.UUID, reader io.Reader, filename string) (string, int64, string, error) {
	if err := s.EnsureUserStorage(userID); err != nil {
		return "", 0, "", err
	}

	ext := filepath.Ext(filename)
	storageName := fmt.Sprintf("%s%s", uuid.New().String(), ext)
	storagePath := filepath.Join(s.GetUserStoragePath(userID), storageName)

	file, err := os.Create(storagePath)
	if err != nil {
		return "", 0, "", err
	}
	defer file.Close()

	hasher := sha256.New()
	writer := io.MultiWriter(file, hasher)

	size, err := io.Copy(writer, reader)
	if err != nil {
		os.Remove(storagePath)
		return "", 0, "", err
	}

	checksum := hex.EncodeToString(hasher.Sum(nil))
	return storagePath, size, checksum, nil
}

func (s *StorageService) DeleteFile(storagePath string) error {
	if storagePath == "" {
		return nil
	}
	return os.Remove(storagePath)
}

func (s *StorageService) GetFile(storagePath string) (*os.File, error) {
	return os.Open(storagePath)
}

func (s *StorageService) GetMimeType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		return "application/octet-stream"
	}
	return mimeType
}

func (s *StorageService) CopyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

func (s *StorageService) CreateFileVersion(file *models.File) error {
	version := &models.FileVersion{
		FileID:      file.ID,
		Version:     file.Version,
		Size:        file.Size,
		StoragePath: file.StoragePath,
		Checksum:    file.Checksum,
	}

	return database.DB.Create(version).Error
}

func (s *StorageService) GetStorageUsage(userID uuid.UUID) (int64, error) {
	var totalSize int64
	err := database.DB.Model(&models.File{}).
		Where("owner_id = ? AND is_directory = false AND is_trashed = false", userID).
		Select("COALESCE(SUM(size), 0)").
		Scan(&totalSize).Error
	return totalSize, err
}

func (s *StorageService) CleanupTrash(userID uuid.UUID, olderThanDays int) error {
	return database.DB.
		Where("owner_id = ? AND is_trashed = true AND trashed_at < NOW() - INTERVAL '? days'", userID, olderThanDays).
		Delete(&models.File{}).Error
}
