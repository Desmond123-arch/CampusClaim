package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	// ID           uint      `gorm:"primaryKey;column:id;autoIncrement"`
	Password     string    `json:"password" gorm:"column:password;not null" validate:"required"`
	FullName     string    `json:"full_name" gorm:"column:full_name;not null" validate:"required"`
	PhoneNumber  string    `json:"phone_number" gorm:"column:phone_number;not null" validate:"required"`
	ImageURL string    `json:"profile_image" gorm:"column:profile_image"`
	RefreshToken string    `gorm:"column:refresh_token"`
	IsVerified bool `json:"is_verified" gorm:"column:is_verified"`
	EmailVerification EmailVerification
}

type EmailVerification struct {
	gorm.Model
	Code string `json:"verification_code" gorm:"column:verification_code;not null"`
	ExpiresAt time.Time `json:"expires_at" gorm:"column:expires_at"`
	UserID uint
}
type Item_Status struct {
	gorm.Model
	Name string `gorm:"unique;column:status"`
}

type Categories struct {
	gorm.Model
	NAME string `gorm:"unique;column:category"`
}



type Item struct {
	gorm.Model
	ItemUUID     uuid.UUID `json:"" gorm:"type:uuid;default:uuid_generate_v4();column:item_uuid"`
	Title string `gorm:"size:1000;column:title;not null" validate:"required"`
	Description string `json:"description" gorm:"size:65535;column:description" validate:"required"`
	Bounty uint `json:"bounty" gorm:"column:bounty;default:0" validate:"required,numeric"`
	UserID uint `gorm:"column:posted_by"`
	StatusID uint `gorm:"column:status_id"`
	CategoryID uint `gorm:"column:category_id"`
	//CategoryID

	User User `gorm:"foreignKey:UserID;references:ID;OnDelete:CASCADE;"`
	Item_Status Item_Status `gorm:"foreignKey:StatusID;references:ID"`
	Categories Categories `gorm:"foreignKey:CategoryID;references:ID"`
}

type Images struct {
	gorm.Model
	ItemID uint `gorm:"column:item_id"`
	ImageUrl string `gorm:"column:image_url;not null"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`

	Item Item `gorm:"foreignKey:ItemID;references:ID"`
}




func Setup(db *gorm.DB) {
	db.AutoMigrate(&User{}, &Item_Status{}, &Categories{}, &Item{}, &Images{}, &EmailVerification{})
}
