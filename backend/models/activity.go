package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ActivityType string

const (
	ActivityFileCreated    ActivityType = "file_created"
	ActivityFileUpdated    ActivityType = "file_updated"
	ActivityFileDeleted    ActivityType = "file_deleted"
	ActivityFileMoved      ActivityType = "file_moved"
	ActivityFileShared     ActivityType = "file_shared"
	ActivityFileDownloaded ActivityType = "file_downloaded"
	ActivityFolderCreated  ActivityType = "folder_created"
	ActivityUserLogin      ActivityType = "user_login"
	ActivityUserLogout     ActivityType = "user_logout"
)

type Activity struct {
	ID        uuid.UUID    `gorm:"type:uuid;primary_key" json:"id"`
	UserID    uuid.UUID    `gorm:"type:uuid;not null;index" json:"user_id"`
	Type      ActivityType `gorm:"type:varchar(50);not null" json:"type"`
	FileID    *uuid.UUID   `gorm:"type:uuid;index" json:"file_id,omitempty"`
	FileName  string       `gorm:"size:255" json:"file_name,omitempty"`
	Details   string       `gorm:"type:text" json:"details,omitempty"`
	IPAddress string       `gorm:"size:45" json:"ip_address,omitempty"`
	UserAgent string       `gorm:"size:500" json:"user_agent,omitempty"`
	CreatedAt time.Time    `json:"created_at"`

	User User  `gorm:"foreignKey:UserID" json:"-"`
	File *File `gorm:"foreignKey:FileID" json:"-"`
}

func (a *Activity) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}
