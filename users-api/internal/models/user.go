package models

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID             int32          `json:"id" gorm:"primaryKey;autoIncrement"`
	Username       string         `json:"username" gorm:"uniqueIndex;not null;size:50"`
	Email          string         `json:"email" gorm:"uniqueIndex;not null;size:100"`
	PasswordHash   string         `json:"-" gorm:"not null;size:255"`
	FirstName      *string        `json:"first_name" gorm:"size:50"`
	LastName       *string        `json:"last_name" gorm:"size:50"`
	Role           UserRole       `json:"role" gorm:"type:enum('normal','admin');default:'normal'"`
	InitialBalance float64        `json:"initial_balance" gorm:"type:decimal(15,2);default:100000.00"`
	CurrentBalance float64        `json:"current_balance" gorm:"-"` // Calculated field, not stored in DB
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `json:"-" gorm:"index"`
	LastLogin      *time.Time     `json:"last_login"`
	IsActive       bool           `json:"is_active" gorm:"default:true"`
	Preferences    string         `json:"preferences" gorm:"type:json"`
}

type UserRole string

const (
	RoleNormal UserRole = "normal"
	RoleAdmin  UserRole = "admin"
)

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.Preferences == "" {
		defaultPrefs := map[string]interface{}{
			"theme":         "light",
			"notifications": true,
			"language":      "en",
		}
		prefsJSON, _ := json.Marshal(defaultPrefs)
		u.Preferences = string(prefsJSON)
	}
	return nil
}

func (u *User) TableName() string {
	return "users"
}

func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

func (u *User) GetFullName() string {
	if u.FirstName != nil && u.LastName != nil {
		return *u.FirstName + " " + *u.LastName
	}
	if u.FirstName != nil {
		return *u.FirstName
	}
	if u.LastName != nil {
		return *u.LastName
	}
	return u.Username
}

func (u *User) GetPreferences() (map[string]interface{}, error) {
	if u.Preferences == "" {
		return map[string]interface{}{
			"theme":         "light",
			"notifications": true,
			"language":      "en",
		}, nil
	}

	var prefs map[string]interface{}
	err := json.Unmarshal([]byte(u.Preferences), &prefs)
	return prefs, err
}

func (u *User) SetPreferences(prefs map[string]interface{}) error {
	if prefs == nil {
		u.Preferences = ""
		return nil
	}

	prefsJSON, err := json.Marshal(prefs)
	if err != nil {
		return err
	}
	u.Preferences = string(prefsJSON)
	return nil
}

type RefreshToken struct {
	ID        int32     `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID    int32     `json:"user_id" gorm:"not null;index"`
	Token     string    `json:"token" gorm:"uniqueIndex;not null;size:500"`
	ExpiresAt time.Time `json:"expires_at" gorm:"not null"`
	CreatedAt time.Time `json:"created_at"`
	Revoked   bool      `json:"revoked" gorm:"default:false"`
	User      User      `json:"-" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}

func (rt *RefreshToken) TableName() string {
	return "refresh_tokens"
}

func (rt *RefreshToken) IsExpired() bool {
	return time.Now().After(rt.ExpiresAt)
}

type LoginAttempt struct {
	ID          int32     `json:"id" gorm:"primaryKey;autoIncrement"`
	Email       string    `json:"email" gorm:"size:100;index"`
	IPAddress   string    `json:"ip_address" gorm:"size:45;index"`
	UserAgent   string    `json:"user_agent" gorm:"type:text"`
	Success     bool      `json:"success"`
	AttemptedAt time.Time `json:"attempted_at" gorm:"autoCreateTime"`
}

func (la *LoginAttempt) TableName() string {
	return "login_attempts"
}
