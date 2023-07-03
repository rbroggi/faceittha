package model

import (
	"time"

	"github.com/google/uuid"
)

// CreateUserArgs contain the arguments of the CreateUser method.
type CreateUserArgs struct {
	// FirstName is the user first name.
	FirstName string

	// LastName is the user last name.
	LastName string

	// Nickname is the user nickname
	Nickname string

	// Email is the user email
	Email string

	// Password is the user password.
	Password string

	// Country is the user country
	Country string
}

// CreateUserResponse contains the response of the CreateUser method.
type CreateUserResponse struct {
	// User
	User User
}

// ListUsersArgs contain the arguments for the ListUsers use-case.
type ListUsersArgs struct {
	// ID is the user-id to query. Zero-value will be ignored as filter.
	ID uuid.UUID

	// Countries to which the desired users belong to. Zero-value will be ignored as filter.
	Countries []string

	// CreatedAfter is the left time boundary in which the user was created. Zero-value will be ignored as filter.
	CreatedAfter time.Time

	// CreatedBefore is the right time boundary in which the user was created. Zero-value will be ignored as filter.
	CreatedBefore time.Time

	// Limit is the maximum amount of users to return (for pagination). Zero-value will be interpreted as no-limit.
	Limit uint32

	// Offset is the offset to apply (for pagination). Zero-value will be interpreted as 0 Offset.
	Offset uint32
}

// ListUsersResponse contains the users matching the input query of the ListUsers api.
type ListUsersResponse struct {
	// Users are the users matching the ListUsers query.
	Users []User
}

// DeleteUserArgs contains the arguments for deleting a user.
type DeleteUserArgs struct {
	// ID is the id of the user to be deleted.
	ID string

	// HardDelete instructs the deletion to be a hard deletion (true). Otherwise, a soft-copy will be kept.
	HardDelete bool
}

// UpdateUserArgs contain the arguments of the UpdateUser method.
type UpdateUserArgs struct {
	// ID is the id of the user to be updated.
	ID string

	// FirstName is the user first name.
	FirstName string

	// LastName is the user last name.
	LastName string

	// Nickname is the user nickname
	Nickname string

	// Email is the user email
	Email string

	// Country is the user country
	Country string
}

// UpdateUserResponse contains the response of the UpdateUser method.
type UpdateUserResponse struct {
	// User
	User User
}
