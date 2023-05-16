//go:build component
// +build component

package component

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/go-pg/pg/v10"
	"github.com/golang/protobuf/proto"
	v1 "github.com/rbroggi/faceittha/pkg/sdk/v1"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
)

// ComponentTestSuite is the test suite gathering structs and utilities for running the component tests.
type ComponentTestSuite struct {
	suite.Suite
	db         *pg.DB
	userClient v1.UserServiceClient
	
	ctx context.Context
	cnl context.CancelFunc
	pubsubClient *pubsub.Client
	wg *sync.WaitGroup
	events <-chan v1.UserEvent

	// internal state persisted cross method calls
	createUserRequest *v1.CreateUserRequest
	createUserResponse *v1.CreateUserResponse

	updateUserRequest *v1.UpdateUserRequest
	updateUserResponse *v1.UpdateUserResponse

	deleteUserRequest *v1.RemoveUserRequest
	deleteUserResponse *v1.RemoveUserResponse
}

func (s *ComponentTestSuite) SetupTest() {
	_, err := s.db.Exec("TRUNCATE TABLE faceittha.users")
	s.Require().NoError(err)
}

func (s *ComponentTestSuite) TearDownSuite() {
	// close the database connection after each test
	s.Require().NoError(s.db.Close())
	s.pubsubClient.Close()
	s.cnl()
	s.wg.Wait()
}

func TestComponentTestSuite(t *testing.T) {
	postgresUrl := os.Getenv("POSTGRESQL_URL")
	if postgresUrl == "" {
		postgresUrl = "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
	}

	grpcServerAddress := os.Getenv("GRPC_SERVER_URL")
	if grpcServerAddress == "" {
		grpcServerAddress = "localhost:50051"
	}

	projectID := os.Getenv("PUBSUB_PROJECT_ID")
	if projectID == "" {
		projectID = "faceittha"
	}
	userPublicSubscriptionID := os.Getenv("PUBSUB_TEST_USER_PUBLIC_EVENT_SUBSCRIPTION_ID")
	if userPublicSubscriptionID == "" {
		userPublicSubscriptionID = "test.shared.facittha.UserEvents.sub"
	}
	emulatorAddr := os.Getenv("PUBSUB_EMULATOR_HOST")
	if emulatorAddr == "" {
		require.NoError(t, os.Setenv("PUBSUB_EMULATOR_HOST", "localhost:8085"))
	}

	// Postgres connection (only for cleaning up data between tests)
	opts, err := pg.ParseURL(postgresUrl)
	require.NoError(t, err)
	db := pg.Connect(opts)
	require.NoError(t, db.Ping(context.Background()))

	// Create a gRPC connection to the server
	conn, err := grpc.Dial(grpcServerAddress, grpc.WithInsecure())
	require.NoError(t, err)

	userSvcClient := v1.NewUserServiceClient(conn)

	// pubsub consumer of public events
	ctx, cnl := context.WithCancel(context.Background())
	client, err := pubsub.NewClient(ctx, projectID)
	require.NoError(t, err)
	wg := &sync.WaitGroup{}
	ch := make(chan v1.UserEvent, 10)
	wg.Add(1)
	go func() {
		defer func() {
			close(ch)
			wg.Done()
		}()
		subscription := client.Subscription(userPublicSubscriptionID)
		subscription.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
			var userEvent v1.UserEvent
			require.NoError(t, proto.Unmarshal(msg.Data, &userEvent))
			ch <- userEvent
			msg.Ack()
		})
	} ()

	suite.Run(t, &ComponentTestSuite{
		db:                 db,
		userClient:         userSvcClient,
		ctx:                ctx,
		cnl:                cnl,
		pubsubClient:       client,
		wg:                 wg,
		events:             ch,
	})
}

type given = func() *ComponentTestSuite
type when = func() *ComponentTestSuite
type then = func() *ComponentTestSuite

func (s *ComponentTestSuite) gherkin() (given, when, then) {
	return func() *ComponentTestSuite { return s}, func() *ComponentTestSuite { return s}, func() *ComponentTestSuite { return s}
}


func (s *ComponentTestSuite) aCreateUserRequestIsIssued() *ComponentTestSuite {
	var err error
	s.createUserRequest = &v1.CreateUserRequest{
		FirstName: "Joe",
		LastName:  "Doe",
		Nickname:  "JD",
		Password:  "SuperSecret",
		Email:     "joeDoe@example.com",
		Country:   "US",
	}
	s.createUserResponse, err = s.userClient.CreateUser(context.Background(), s.createUserRequest)
	s.Require().NoError(err)

	return s
}

func (s *ComponentTestSuite) theUserGetsUpdated() *ComponentTestSuite {
	var err error
	s.updateUserRequest = &v1.UpdateUserRequest{
		Id:        s.createUserResponse.User.Id,
		FirstName: s.createUserResponse.User.FirstName + "update",
	}
	s.updateUserResponse, err = s.userClient.UpdateUser(context.Background(), s.updateUserRequest)
	s.Require().NoError(err)
	return s
}

func (s *ComponentTestSuite) aUserDeletionRequestIsIssued() *ComponentTestSuite {
	var err error
	s.deleteUserRequest = &v1.RemoveUserRequest{
		Id:         s.createUserResponse.User.Id,
	}
	s.deleteUserResponse, err = s.userClient.RemoveUser(context.Background(), s.deleteUserRequest)
	s.Require().NoError(err)
	return s

}

func (s *ComponentTestSuite) theUpdateResponseReflectsTheUpdateOperation() *ComponentTestSuite {
	s.Require().NotNil(s.updateUserResponse)
	s.Require().Equal(s.updateUserResponse.User.Id, s.createUserResponse.User.Id)
	s.Require().NotEqual(s.updateUserResponse.User.FirstName, s.createUserResponse.User.FirstName)
	s.Require().Equal(s.updateUserResponse.User.LastName, s.createUserResponse.User.LastName)
	return s
}

func (s *ComponentTestSuite) anExistingUser() *ComponentTestSuite {
	return s.aCreateUserRequestIsIssued().
			theCreateUserResponseContainsAValidUser()
}

func (s *ComponentTestSuite) theCreateUserResponseContainsAValidUser() *ComponentTestSuite {
	s.Require().NotNil(s.createUserResponse)
	
	s.Require().Equal(s.createUserRequest.FirstName, s.createUserResponse.User.FirstName)
	s.Require().Equal(s.createUserRequest.LastName, s.createUserResponse.User.LastName)
	s.Require().Equal(s.createUserRequest.Nickname, s.createUserResponse.User.Nickname)
	s.Require().Equal(s.createUserRequest.Email, s.createUserResponse.User.Email)
	s.Require().Equal(s.createUserRequest.Country, s.createUserResponse.User.Country)
	s.Require().NotEmpty(s.createUserResponse, s.createUserResponse.User.Id)
	
	return s
}


func (s *ComponentTestSuite) listUsersContainsTheCreatedUser() *ComponentTestSuite {
	listUsersResp, err := s.userClient.ListUsers(context.Background(), &v1.ListUsersRequest{})
	s.Require().NoError(err)
	var containsCreatedUser bool
	for _, u := range listUsersResp.Users {
		if u.Id == s.createUserResponse.User.Id {
			containsCreatedUser = true
		}
	}
	s.Require().True(containsCreatedUser)
	return s
}

func (s *ComponentTestSuite) listUsersContainsTheUpdatedUser() *ComponentTestSuite {
	listUsersResp, err := s.userClient.ListUsers(context.Background(), &v1.ListUsersRequest{})
	s.Require().NoError(err)
	var containsCreatedUser bool
	for _, u := range listUsersResp.Users {
		if u.Id == s.createUserResponse.User.Id {
			s.Require().NotEqual(u.FirstName, s.createUserResponse.User.FirstName)
			s.Require().Equal(u.FirstName, s.updateUserResponse.User.FirstName)
			containsCreatedUser = true
		}
	}
	s.Require().True(containsCreatedUser)
	return s
}

func (s *ComponentTestSuite) listUsersDoesNotContainTheUser() *ComponentTestSuite {
	listUsersResp, err := s.userClient.ListUsers(context.Background(), &v1.ListUsersRequest{})
	s.Require().NoError(err)
	var containsCreatedUser bool
	for _, u := range listUsersResp.Users {
		if u.Id == s.deleteUserRequest.Id {
			containsCreatedUser = true
		}
	}
	s.Require().False(containsCreatedUser)
	return s

}

func (s *ComponentTestSuite) anEventForTheUserCreationWillEventuallyBeProduced() *ComponentTestSuite {
	timeoutCh := time.After(time.Second * 5)
	for {
		select {
		case event, more := <-s.events:
			if !more {
				s.Fail("channel closed before reaching desired event")
			}

			// success
			if event.Before == nil && event.After != nil && event.After.Id == s.createUserResponse.User.Id {
				return s
			}

		case <-timeoutCh:
			// Timeout occurred
			s.Fail("timeout before receiving creation event")
		}
	}
}

func (s *ComponentTestSuite) anEventForTheUserUpdateWillEventuallyBeProduced() *ComponentTestSuite {
	timeoutCh := time.After(time.Second * 5)
	for {
		select {
		case event, more := <-s.events:
			if !more {
				s.Fail("channel closed before reaching desired event")
			}

			// success
			if event.Before != nil && event.After != nil && event.After.Id == s.updateUserResponse.User.Id {
				s.Require().Equal(event.Before.FirstName, s.createUserResponse.User.FirstName)
				s.Require().NotEqual(event.Before.FirstName, s.updateUserResponse.User.FirstName)
				s.Require().Equal(event.After.FirstName, s.updateUserResponse.User.FirstName)
				s.Require().NotEqual(event.After.FirstName, s.createUserResponse.User.FirstName)
				return s
			}

		case <-timeoutCh:
			// Timeout occurred
			s.Fail("timeout before receiving creation event")
		}
	}

}

func (s *ComponentTestSuite) anEventForTheUserDeletionWillEventuallyBeProduced() *ComponentTestSuite {
	timeoutCh := time.After(time.Second * 5)
	for {
		select {
		case event, more := <-s.events:
			if !more {
				s.Fail("channel closed before reaching desired event")
			}

			// success
			if event.Before != nil && event.After == nil && event.Before.Id == s.deleteUserRequest.Id {
				return s
			}

		case <-timeoutCh:
			// Timeout occurred
			s.Fail("timeout before receiving creation event")
		}
	}
}