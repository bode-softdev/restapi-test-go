package model

import (
	"time"

	"gorm.io/datatypes"
)

type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"size:30;uniqueIndex;not null" json:"username"`
	PasswordHash string    `gorm:"size:255;not null" json:"-"`
	Role         string    `gorm:"size:20;not null;default:user" json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Product struct {
	ID          uint64                      `gorm:"primaryKey" json:"id"`
	Title       string                      `gorm:"size:191;not null" json:"title"`
	Price       float64                     `gorm:"type:decimal(12,2);not null" json:"price"`
	Description string                      `gorm:"type:text" json:"description"`
	Category    string                      `gorm:"size:100;not null;index" json:"category"`
	Images      datatypes.JSONSlice[string] `gorm:"type:jsonb" json:"images"`

	CreatedAt   time.Time `json:"created_at"`
	CreatedBy   string    `gorm:"size:100" json:"created_by"`
	CreatedByID uint      `json:"created_by_id"`
	UpdatedAt   time.Time `json:"updated_at"`
	UpdatedBy   string    `gorm:"size:100" json:"updated_by"`
	UpdatedByID uint      `json:"updated_by_id"`
}

type RefreshToken struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	UserID    uint   `gorm:"not null;index" json:"user_id"`
	TokenHash string `gorm:"size:64;uniqueIndex;not null" json:"-"`
	AccessJTI string `gorm:"size:64;index" json:"-"`

	ExpiresAt time.Time  `gorm:"not null;index" json:"expires_at"`
	RevokedAt *time.Time `json:"revoked_at"`

	AccessRevokedAt *time.Time `json:"access_revoked_at"`
	CreatedAt       time.Time  `json:"created_at"`
}
