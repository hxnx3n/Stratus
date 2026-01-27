package models

import (
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type File struct {
	ID          uuid.UUID      `gorm:"type:uuid;primary_key" json:"id"`
	Name        string         `gorm:"not null;size:255" json:"name"`
	Path        string         `gorm:"not null" json:"path"`
	StoragePath string         `gorm:"not null" json:"-"`
	MimeType    string         `gorm:"size:100" json:"mime_type"`
	Size        int64          `gorm:"default:0" json:"size"`
	IsDirectory bool           `gorm:"default:false" json:"is_directory"`
	ParentID    *uuid.UUID     `gorm:"type:uuid;index" json:"parent_id"`
	OwnerID     uuid.UUID      `gorm:"type:uuid;not null;index" json:"owner_id"`
	Checksum    string         `gorm:"size:64" json:"checksum"`
	Version     int            `gorm:"default:1" json:"version"`
	IsTrashed   bool           `gorm:"default:false" json:"is_trashed"`
	TrashedAt   *time.Time     `json:"trashed_at,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	Owner    User   `gorm:"foreignKey:OwnerID" json:"-"`
	Parent   *File  `gorm:"foreignKey:ParentID" json:"-"`
	Children []File `gorm:"foreignKey:ParentID" json:"children,omitempty"`
}

func (f *File) BeforeCreate(tx *gorm.DB) error {
	if f.ID == uuid.Nil {
		f.ID = uuid.New()
	}
	return nil
}

func (f *File) GetExtension() string {
	if f.IsDirectory {
		return ""
	}
	return filepath.Ext(f.Name)
}

func (f *File) GetFullPath() string {
	return filepath.Join(f.Path, f.Name)
}

type FileVersion struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	FileID      uuid.UUID `gorm:"type:uuid;not null;index" json:"file_id"`
	Version     int       `gorm:"not null" json:"version"`
	Size        int64     `gorm:"default:0" json:"size"`
	StoragePath string    `gorm:"not null" json:"-"`
	Checksum    string    `gorm:"size:64" json:"checksum"`
	CreatedAt   time.Time `json:"created_at"`

	File File `gorm:"foreignKey:FileID" json:"-"`
}

func (fv *FileVersion) BeforeCreate(tx *gorm.DB) error {
	if fv.ID == uuid.Nil {
		fv.ID = uuid.New()
	}
	return nil
}
