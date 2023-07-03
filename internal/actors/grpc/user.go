package grpc

import (
	"context"
	"errors"
	"time"

	"github.com/rbroggi/faceittha/internal/core/model"
	pb "github.com/rbroggi/faceittha/pkg/sdk/v1"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// UserServiceArgs are the mandatory args to instantiate the UserService.
type UserServiceArgs struct {
	// Usecase is the usecase for user-service
	Usecase userServiceUsecase
}

// NewUserService creates a new UserService
func NewUserService(args UserServiceArgs) *UserService {
	return &UserService{usecase: args.Usecase}
}

// UserService implements the User service gRPC methods.
type UserService struct {
	pb.UnimplementedUserServiceServer
	usecase userServiceUsecase
}

func (u *UserService) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	resp, err := u.usecase.CreateUser(ctx, model.CreateUserArgs{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Nickname:  req.Nickname,
		Email:     req.Email,
		Password:  req.Password,
		Country:   req.Country,
	})
	if err != nil {
		log.WithError(err).Error("error invoking usecase CreateUser")
		return nil, status.Errorf(codes.Internal, "internal error")
	}

	return &pb.CreateUserResponse{
		User: &pb.User{
			Id:        resp.User.ID,
			FirstName: resp.User.FirstName,
			LastName:  resp.User.LastName,
			Nickname:  resp.User.Nickname,
			Email:     resp.User.Email,
			Country:   resp.User.Country,
			CreatedAt: timestamppb.New(resp.User.CreatedAt),
			UpdatedAt: timestamppb.New(resp.User.UpdatedAt),
		},
	}, nil
}

func (u *UserService) ListUsers(ctx context.Context, req *pb.ListUsersRequest) (*pb.ListUsersResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	createdAfter := time.Time{}
	if req.CreatedAfter.IsValid() {
		createdAfter = req.CreatedAfter.AsTime()
	}
	createdBefore := time.Time{}
	if req.CreatedBefore.IsValid() {
		createdBefore = req.CreatedBefore.AsTime()
	}

	resp, err := u.usecase.ListUsers(ctx, model.ListUsersArgs{
		Countries:     req.Countries,
		CreatedAfter:  createdAfter,
		CreatedBefore: createdBefore,
		Limit:         req.GetPageSize(),
		Offset:        req.GetOffset(),
	})
	if err != nil {
		log.WithError(err).Error("error invoking usecase ListUsers")
		return nil, status.Errorf(codes.Internal, "internal error")
	}

	return &pb.ListUsersResponse{Users: usersToProto(resp.Users)}, nil
}

// UpdateUser updates a user.
func (u *UserService) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.UpdateUserResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	updateResp, err := u.usecase.UpdateUser(ctx, model.UpdateUserArgs{
		ID:        req.Id,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Nickname:  req.Nickname,
		Email:     req.Email,
		Country:   req.Country,
	})
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			log.Warn("attempt to update non-existing user")
			return nil, status.Errorf(codes.NotFound, "user not found")
		}

		log.WithError(err).Error("error invoking usecase UpdateUser")
		return nil, status.Errorf(codes.Internal, "internal error")
	}

	return &pb.UpdateUserResponse{
		User: &pb.User{
			Id:        updateResp.User.ID,
			FirstName: updateResp.User.FirstName,
			LastName:  updateResp.User.LastName,
			Nickname:  updateResp.User.Nickname,
			Email:     updateResp.User.Email,
			Country:   updateResp.User.Country,
			CreatedAt: timestamppb.New(updateResp.User.CreatedAt),
			UpdatedAt: timestamppb.New(updateResp.User.UpdatedAt),
		},
	}, nil
}

// RemoveUser deletes a user.
func (u *UserService) RemoveUser(ctx context.Context, req *pb.RemoveUserRequest) (*pb.RemoveUserResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	if err := u.usecase.DeleteUser(ctx, model.DeleteUserArgs{
		ID:         req.Id,
		HardDelete: req.HardDelete,
	}); err != nil {
		log.WithError(err).Error("error invoking usecase RemoveUser")
		return nil, status.Errorf(codes.Internal, "internal error")
	}

	return &pb.RemoveUserResponse{}, nil
}

// userServiceUsecase
type userServiceUsecase interface {
	// CreateUser creates a user.
	CreateUser(ctx context.Context, args model.CreateUserArgs) (*model.CreateUserResponse, error)

	// UpdateUser updates a user.
	UpdateUser(ctx context.Context, args model.UpdateUserArgs) (*model.UpdateUserResponse, error)

	// ListUsers lists users.
	ListUsers(ctx context.Context, args model.ListUsersArgs) (*model.ListUsersResponse, error)

	// DeleteUser deletes a user.
	DeleteUser(ctx context.Context, args model.DeleteUserArgs) error
}

func usersToProto(users []model.User) []*pb.User {
	ret := make([]*pb.User, len(users))
	for i, u := range users {
		ret[i] = userToProto(u)
	}
	return ret
}

func userToProto(user model.User) *pb.User {
	return &pb.User{
		Id:        user.ID,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Nickname:  user.Nickname,
		Email:     user.Email,
		Country:   user.Country,
		CreatedAt: timestamppb.New(user.CreatedAt),
		UpdatedAt: timestamppb.New(user.UpdatedAt),
	}
}
