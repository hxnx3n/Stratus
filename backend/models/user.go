package models

import (
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type User struct {
	ID           uuid.UUID      `gorm:"type:uuid;primary_key" json:"id"`
	Username     string         `gorm:"uniqueIndex;not null;size:100" json:"username"`
	Email        string         `gorm:"uniqueIndex;not null;size:255" json:"email"`
	PasswordHash string         `gorm:"not null" json:"-"`
	DisplayName  string         `gorm:"size:255" json:"display_name"`
	Quota        int64          `gorm:"default:10737418240" json:"quota"`
	UsedSpace    int64          `gorm:"default:0" json:"used_space"`
	IsAdmin      bool           `gorm:"default:false" json:"is_admin"`
	IsActive     bool           `gorm:"default:true" json:"is_active"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`

	Files []File `gorm:"foreignKey:OwnerID" json:"-"`
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

func (u *User) SetPassword(password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.PasswordHash = string(hash)
	return nil
}

func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil
}

func (u *User) HasEnoughSpace(size int64) bool {
	return u.UsedSpace+size <= u.Quota
}
