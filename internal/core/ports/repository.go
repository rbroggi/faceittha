package ports

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/rbroggi/faceittha/internal/core/model"
)

// Repository is the interface for the persistence layer.
type Repository interface {
	// SaveUser durably saves the user.
	SaveUser(ctx context.Context, user *model.User) error

	// UpdateUser updates the user and saves the state in the persistence layer.
	// All the non-zero values specified will be updated.
	UpdateUser(ctx context.Context, user *model.User) error

	// ListUsers lists all users matching the query parameters.
	ListUsers(ctx context.Context, query ListUsersQuery) (*ListUsersResult, error)

	// DeleteUser removes the user matching the query parameters.
	DeleteUser(ctx context.Context, query DeleteUserQuery) error
}

// ListUsersQuery gather the parameters for which the query
type ListUsersQuery struct {
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

// ListUsersResult gathers the result
type ListUsersResult struct {
	// Users are the users matching the query parameters
	Users []model.User
}

// DeleteUserQuery
type DeleteUserQuery struct {
	// ID is the ID of the user to be deleted
	ID string

	// HardDelete will hard-delete the user, otherwise it's kept in soft-delete state for auditing
	HardDelete bool
}
