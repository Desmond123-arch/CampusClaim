package models

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	// ID           uint      `gorm:"primaryKey;column:id;autoIncrement"`
	UUID               uuid.UUID         `gorm:"type:uuid;default:uuid_generate_v4();"`
	Password           string            `json:"password,omitempty" gorm:"column:password;not null" validate:"required,validate_password"`
	ConfirmPassword    string            `json:"confirm_password,omitempty" gorm:"-" validate:"required"`
	FullName           string            `json:"full_name" gorm:"column:full_name;not null" validate:"required"`
	Email              string            `json:"email" gorm:"column:email;not null;unique" validate:"required,email,school_email"`
	PhoneNumber        string            `json:"phone_number" gorm:"column:phone_number;not null" validate:"required"`
	ImageURL           string            `json:"profile_image" gorm:"column:profile_image"`
	RefreshToken       string            `gorm:"column:refresh_token"`
	PasswordResetToken string            `gorm:"column:password_token;"`
	IsVerified         bool              `json:"is_verified" gorm:"column:is_verified"`
	EmailVerification  EmailVerification `json:"omitempty"`
}

func (u *User) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ID          uuid.UUID `json:"id"`
		FullName    string    `json:"full_name"`
		Email       string    `json:"email"`
		PhoneNumber string    `json:"phone_number"`
		ImageURL    string    `json:"profile_image"`
		IsVerified  bool      `json:"is_verified"`
	}{
		ID:          u.UUID,
		FullName:    u.FullName,
		Email:       u.Email,
		PhoneNumber: u.PhoneNumber,
		ImageURL:    u.ImageURL,
		IsVerified:  u.IsVerified,
	})
}

type LoginDetails struct {
	Email    string `json:"email" validate:"required,email,school_email"`
	Password string `json:"password" validate:"required"`
}

type EmailVerification struct {
	gorm.Model
	Code      string    `json:"verification_code" gorm:"column:verification_code;not null"`
	ExpiresAt time.Time `json:"expires_at" gorm:"column:expires_at"`
	UserID    uint
}

type Item_Status struct {
	gorm.Model
	Name string `gorm:"unique;column:status"`
}
type Claim_Status struct {
	gorm.Model
	Name string `gorm:"unique;column:status"`
}

type Categories struct {
	gorm.Model
	NAME string `gorm:"unique;column:category"`
}

type Item struct {
	gorm.Model
	UUID        uuid.UUID `json:"item_uuid" gorm:"type:uuid;default:uuid_generate_v4();column:item_uuid"`
	Title       string    `json:"title" gorm:"size:1000;column:title;not null" validate:"required"`
	Description string    `json:"description" gorm:"size:65535;column:description" validate:"required"`
	Bounty      uint      `json:"bounty" gorm:"column:bounty;default:0" validate:"required,numeric"`

	UserID     uint `gorm:"column:posted_by"`
	StatusID   uint `gorm:"column:status_id"`
	CategoryID uint `gorm:"column:category_id"`

	User        User        `gorm:"foreignKey:UserID;references:ID;OnDelete:CASCADE"`
	Item_Status Item_Status `gorm:"foreignKey:StatusID;references:ID"`
	Categories  Categories  `gorm:"foreignKey:CategoryID;references:ID"`
	Images      []Images    `gorm:"foreignKey:ItemID;references:ID"` // Assuming you want multiple images per item
}

func (i Item) MarshalJSON() ([]byte, error) {
	// Collect image URLs
	imageURLs := []string{}
	for _, img := range i.Images {
		imageURLs = append(imageURLs, img.ImageUrl)
	}

	return json.Marshal(&struct {
		UUID        uuid.UUID `json:"item_uuid"`
		Title       string    `json:"title"`
		Description string    `json:"description"`
		Bounty      uint      `json:"bounty"`
		Status      string    `json:"status"`
		Category    string    `json:"category"`
		PostedBy    string    `json:"posted_by"`
		CreatedAt   time.Time `json:"created_at"`
		ImageUrls   []string  `json:"image_urls"`
	}{
		UUID:        i.UUID,
		Title:       i.Title,
		Description: i.Description,
		Bounty:      i.Bounty,
		Status:      i.Item_Status.Name,
		Category:    i.Categories.NAME,
		PostedBy:    i.User.FullName,
		CreatedAt:   i.CreatedAt,
		ImageUrls:   imageURLs,
	})
}

type Images struct {
	gorm.Model
	ItemID    uint      `json:"item_id" gorm:"column:item_id"`
	ImageUrl  string    `json:"image_url" gorm:"column:image_url;not null"`
	UpdatedAt time.Time `json:"updated_at" gorm:"column:updated_at"`

	// Avoid deep recursion by omitting Item or making it ignored by JSON
	Item *Item `gorm:"foreignKey:ItemID;references:ID" json:"-"`
}

type Claims struct {
	gorm.Model
	ClaimID  uuid.UUID `gorm:"column:claim_id;default:uuid_generate_v4();"`
	ItemID   uint      `gorm:"column:item_id;uniqueIndex:idx_user_item"`
	UserID   uint      `gorm:"column:posted_by;uniqueIndex:idx_user_item"`
	StatusID uint      `json:"status,omitempty" gorm:"column:status_id"`

	User        User         `gorm:"foreignKey:UserID;references:ID;OnDelete:CASCADE;" validate:"-"`
	ClaimStatus Claim_Status `gorm:"foreignKey:StatusID;references:ID" validate:"-"`
	Item        Item         `gorm:"foreignKey:ItemID;references:ID" validate:"-"`
}

func (c Claims) MarshalJSON() ([]byte, error) {

	return json.Marshal(&struct {
		ID        uint      `json:"id"`
		ClaimID   uuid.UUID `json:"claim_id"`
		ItemID    uint      `json:"item_id"`
		UserID    uint      `json:"user_id"`
		Status    string    `json:"status"`
		ClaimedBy string    `json:"claimed_by"`
		ItemTitle string    `json:"item_title"`
		ItemUUID  uuid.UUID `json:"item_uuid"`
		Bounty    uint      `json:"bounty"`
		CreatedAt time.Time `json:"created_at"`
	}{
		ID:        c.ID,
		ClaimID:   c.ClaimID,
		ItemID:    c.ItemID,
		UserID:    c.UserID,
		Status:    c.ClaimStatus.Name,
		ClaimedBy: c.User.FullName,
		ItemTitle: c.Item.Title,
		ItemUUID:  c.Item.UUID,
		Bounty:    c.Item.Bounty,
		CreatedAt: c.CreatedAt,
	})
}

func Setup(db *gorm.DB) {
	fmt.Printf("CREATING TABLES")
	db.AutoMigrate(
		&User{}, &Item_Status{},
		&Claims{}, &Categories{},
		&Item{}, &Images{},
		&EmailVerification{})

	item_status := []Item_Status{
		{Name: "LOST"},
		{Name: "FOUND"},
		{Name: "CLAIMED"},
	}
	claim_status := []Claim_Status{
		{Name: "Pending"},
		{Name: "Approved"},
		{Name: "Rejected"},
	}
	categories := []Categories{
		{NAME: "Electronics"},
		{NAME: "Clothing"},
		{NAME: "Books"},
		{NAME: "Accessories"},
		{NAME: "School Supplies"},
		{NAME: "Documents"},
		{NAME: "Keys"},
		{NAME: "Wallets"},
	}

	for _, i := range item_status {
		db.FirstOrCreate(&i, Item_Status{Name: i.Name})
	}
	for _, c := range claim_status {
		db.FirstOrCreate(&c, Claim_Status{Name: c.Name})
	}
	for _, cat := range categories {
		db.FirstOrCreate(&cat, Categories{NAME: cat.NAME})
	}
}
