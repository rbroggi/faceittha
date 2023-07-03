package usecase

import (
	"context"
	"fmt"

	"github.com/alexedwards/argon2id"
	"github.com/rbroggi/faceittha/internal/core/model"
	"github.com/rbroggi/faceittha/internal/core/ports"
)

// UserServiceArgs contains the mandatory arguments for the UserService.
type UserServiceArgs struct {
	// Repository is the repository for persistance operations.
	Repository ports.Repository
}

// NewUserService creates a new UserService.
func NewUserService(args UserServiceArgs) *UserService {
	return &UserService{repository: args.Repository}
}

// UserService gathers the functionality around the user-lifecycle
type UserService struct {
	repository ports.Repository
}

// CreateUser creates a user.
func (s *UserService) CreateUser(ctx context.Context, args model.CreateUserArgs) (*model.CreateUserResponse, error) {
	// CreateHash returns a Argon2id hash of a plain-text password using the
	// provided algorithm parameters. The returned hash follows the format used
	// by the Argon2 reference C implementation and looks like this:
	// $argon2id$v=19$m=65536,t=3,p=2$c29tZXNhbHQ$RdescudvJCsgt3ub+b+dWRWJTmaaJObG
	hash, err := argon2id.CreateHash(args.Password, argon2id.DefaultParams)
	if err != nil {
		return nil, fmt.Errorf("error creating password hash: %w", err)
	}

	user := &model.User{
		FirstName:    args.FirstName,
		LastName:     args.LastName,
		Nickname:     args.Nickname,
		Email:        args.Email,
		PasswordHash: hash,
		Country:      args.Country,
	}

	if err := s.repository.SaveUser(ctx, user); err != nil {
		return nil, fmt.Errorf("error saving user in repository: %w", err)
	}

	return &model.CreateUserResponse{User: *user}, nil
}

// UpdateUser updates a user. It returns model.ErrNotFound if the ID does not correspond to an existing user.
func (s *UserService) UpdateUser(ctx context.Context, args model.UpdateUserArgs) (*model.UpdateUserResponse, error) {
	user := &model.User{
		ID:        args.ID,
		FirstName: args.FirstName,
		LastName:  args.LastName,
		Nickname:  args.Nickname,
		Email:     args.Email,
		Country:   args.Country,
	}
	if err := s.repository.UpdateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("error updating user: %w", err)
	}
	return &model.UpdateUserResponse{User: *user}, nil
}

// ListUsers lists users matching the arguments.
func (s *UserService) ListUsers(ctx context.Context, args model.ListUsersArgs) (*model.ListUsersResponse, error) {
	res, err := s.repository.ListUsers(ctx, ports.ListUsersQuery{
		ID:            args.ID,
		Countries:     args.Countries,
		CreatedAfter:  args.CreatedAfter,
		CreatedBefore: args.CreatedBefore,
		Limit:         args.Limit,
		Offset:        args.Offset,
	})
	if err != nil {
		return nil, fmt.Errorf("erro listing users on the repository: %w", err)
	}

	return &model.ListUsersResponse{Users: res.Users}, nil
}

// DeleteUser deletes a user matching the input arguments.
func (s *UserService) DeleteUser(ctx context.Context, args model.DeleteUserArgs) error {
	if err := s.repository.DeleteUser(ctx, ports.DeleteUserQuery{
		ID:         args.ID,
		HardDelete: args.HardDelete,
	}); err != nil {
		return fmt.Errorf("error deleting user from repository: %w", err)
	}
	return nil
}
