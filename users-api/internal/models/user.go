package models

import (
	"time"
	"encoding/json"
	"gorm.io/gorm"
)

type User struct {
	ID             uint           `json:"id" gorm:"primaryKey;autoIncrement"`
	Username       string         `json:"username" gorm:"uniqueIndex;not null;size:50"`
	Email          string         `json:"email" gorm:"uniqueIndex;not null;size:100"`
	PasswordHash   string         `json:"-" gorm:"not null;size:255"`
	FirstName      *string        `json:"first_name" gorm:"size:50"`
	LastName       *string        `json:"last_name" gorm:"size:50"`
	Role           UserRole       `json:"role" gorm:"type:enum('normal','admin');default:'normal'"`
	InitialBalance float64        `json:"initial_balance" gorm:"type:decimal(15,2);default:100000.00"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `json:"-" gorm:"index"`
	LastLogin      *time.Time     `json:"last_login"`
	IsActive       bool           `json:"is_active" gorm:"default:true"`
	Preferences    *UserPrefs     `json:"preferences" gorm:"type:json"`
}

type UserRole string

const (
	RoleNormal UserRole = "normal"
	RoleAdmin  UserRole = "admin"
)

type UserPrefs struct {
	Theme         string `json:"theme"`
	Notifications bool   `json:"notifications"`
	Language      string `json:"language"`
}

func (up *UserPrefs) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, up)
	case string:
		return json.Unmarshal([]byte(v), up)
	}
	return nil
}

func (up UserPrefs) Value() (interface{}, error) {
	return json.Marshal(up)
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.Preferences == nil {
		u.Preferences = &UserPrefs{
			Theme:         "light",
			Notifications: true,
			Language:      "en",
		}
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

type RefreshToken struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID    uint      `json:"user_id" gorm:"not null;index"`
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
	ID          uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	Email       string    `json:"email" gorm:"size:100;index"`
	IPAddress   string    `json:"ip_address" gorm:"size:45;index"`
	UserAgent   string    `json:"user_agent" gorm:"type:text"`
	Success     bool      `json:"success"`
	AttemptedAt time.Time `json:"attempted_at" gorm:"default:CURRENT_TIMESTAMP"`
}

func (la *LoginAttempt) TableName() string {
	return "login_attempts"
}