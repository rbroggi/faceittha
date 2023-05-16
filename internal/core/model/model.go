package model

import (
	"time"

	"github.com/google/uuid"
)

// User represents a user in the system.
type User struct {
	// ID unique identifier of the user.
	ID uuid.UUID `json:"id"`

	// FirstName is the user first name.
	FirstName string `json:"first_name"`

	// LastName is the user last name.
	LastName string `json:"last_name"`

	// Nickname is the user nickname
	Nickname string `json:"nickname"`

	// Email is the user email
	Email string `json:"email"`

	// PasswordHash contains the password hash.
	PasswordHash string `json:"password_hash,omitempty"`

	// Country is the user country
	Country string `json:"country"`

	// CreatedAt is the time at which the user was created in the system.
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is the time at which the user was last updated
	UpdatedAt time.Time `json:"updated_at,omitempty"`

	// DeletedAt is the time at which the user was deleted. Zero-valued if user not deleted
	DeletedAt time.Time `json:"deleted_at,omitempty"`
}

// UserEvent collects a user change. It can represent creation, update and deletion of a user.
type UserEvent struct {
	// ID is the event id.
	ID string 

	// Before is the user state before the event. It will be nil in case of user-creations.
	Before *User

	// After is the user state after the event. It will be nil in case of hard-deletions.
	After *User
}
