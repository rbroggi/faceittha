package mongo

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/rbroggi/faceittha/internal/core/model"
	"github.com/rbroggi/faceittha/internal/core/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoDBTestSuite struct {
	suite.Suite
	db             *mongo.Client
	userCollection *mongo.Collection
	mongoAdapter   *MongoDB
}

var (
	dummyTime = time.Now().Truncate(time.Second).UTC()
)

func (suite *MongoDBTestSuite) SetupSuite() {
	url := os.Getenv("MONGODB_URL")
	if url == "" {
		url = "mongodb://mongouser:mongopwd@localhost:27017/faceit?authSource=admin&readPreference=primary&ssl=false&replicaSet=rs0"
	}

	clientOptions := options.Client().ApplyURI(url)
	db, err := mongo.Connect(context.Background(), clientOptions)
	timeoutCtx, _ := context.WithTimeout(context.Background(), 2*time.Second)
	suite.Require().NoError(db.Ping(timeoutCtx, nil))
	collection := db.Database("faceittha").Collection("users")
	dummyTimeFunc := func() time.Time {
		return dummyTime
	}
	mongoAdapter, err := NewMongoDB(MongoDBArgs{UserCollection: collection}, WithNowFunc(dummyTimeFunc))
	suite.Require().NoError(err)
	suite.mongoAdapter = mongoAdapter
	suite.db = db
	suite.userCollection = collection

}

func (suite *MongoDBTestSuite) SetupTest() {
	_, err := suite.db.Database("faceittha").Collection("users").DeleteMany(context.Background(), bson.D{})
	suite.Require().NoError(err)
}

func (suite *MongoDBTestSuite) TearDownSuite() {
	// close the database connection after each test
	suite.Require().NoError(suite.db.Disconnect(context.Background()))
}

func (suite *MongoDBTestSuite) TestSaveUser() {
	tests := []struct {
		name        string
		input       *model.User
		expectedErr assert.ErrorAssertionFunc
		expectedDB  func(input *model.User, collection *mongo.Collection)
	}{
		{
			name: "insert new user",
			input: &model.User{
				ID:           primitive.NewObjectID().Hex(),
				FirstName:    "Jane",
				LastName:     "Doe",
				Nickname:     "jd",
				Email:        "newuser@example.com",
				PasswordHash: "hash",
				Country:      "UK",
			},
			expectedDB: func(input *model.User, collection *mongo.Collection) {
				res := collection.FindOne(context.Background(), bson.D{bson.E{Key: "_id", Value: mustID(suite.T(), input.ID)}})
				suite.NoError(res.Err())
				got := new(userDB)
				suite.Require().NoError(res.Decode(got))
				suite.Equal(got.ID.Hex(), input.ID)
				suite.Equal(got.FirstName, input.FirstName)
				suite.Equal(got.LastName, input.LastName)
				suite.Equal(got.Nickname, input.Nickname)
				suite.Equal(got.Email, input.Email)
				suite.Equal(got.PasswordHash, input.PasswordHash)
				suite.Equal(got.Country, input.Country)
			},
		},
	}

	for _, test := range tests {
		suite.Run(test.name, func() {
			// insert or update the user
			err := suite.mongoAdapter.SaveUser(context.Background(), test.input)
			if test.expectedErr != nil {
				test.expectedErr(suite.T(), err)
			} else {
				suite.Require().NoError(err)
			}
			if test.expectedDB != nil {
				test.expectedDB(test.input, suite.userCollection)
			}
		})
	}
}

func (suite *MongoDBTestSuite) TestUpdateUser() {
	tests := []struct {
		name        string
		existing    *model.User
		input       *model.User
		expectedErr assert.ErrorAssertionFunc
		expectedDB  func(existing, input *model.User, collection *mongo.Collection)
	}{
		{
			name: "save existing user (update)",
			existing: &model.User{
				ID:           mustID(suite.T(), "3b3e9e2a13d54a68b5c58e60").Hex(),
				FirstName:    "Jane",
				LastName:     "Doe",
				Nickname:     "jd",
				Email:        "newuser@example.com",
				PasswordHash: "hash",
				Country:      "UK",
			},
			input: &model.User{
				ID:           mustID(suite.T(), "3b3e9e2a13d54a68b5c58e60").Hex(),
				FirstName:    "Jane2",
				LastName:     "Doe2",
				Nickname:     "jd2",
				Email:        "newuser2@example.com",
				PasswordHash: "hash2",
				Country:      "BR",
			},
			expectedDB: func(existing, input *model.User, collection *mongo.Collection) {
				res := collection.FindOne(context.Background(), bson.D{bson.E{Key: "_id", Value: mustID(suite.T(), input.ID)}})
				suite.NoError(res.Err())
				got := new(userDB)
				suite.Require().NoError(res.Decode(got))
				suite.Equal(got.ID.Hex(), existing.ID)
				suite.Equal(got.FirstName, input.FirstName)
				suite.NotEqual(got.FirstName, existing.FirstName)
				suite.Equal(got.LastName, input.LastName)
				suite.NotEqual(got.LastName, existing.LastName)
				suite.Equal(got.Nickname, input.Nickname)
				suite.NotEqual(got.Nickname, existing.Nickname)
				suite.Equal(got.Email, input.Email)
				suite.NotEqual(got.Email, existing.Email)
				suite.Equal(got.PasswordHash, input.PasswordHash)
				suite.NotEqual(got.PasswordHash, existing.PasswordHash)
				suite.Equal(got.Country, input.Country)
				suite.NotEqual(got.Country, existing.Country)
			},
		},
	}

	for _, test := range tests {
		suite.Run(test.name, func() {
			if test.existing != nil {
				suite.Require().NoError(suite.mongoAdapter.SaveUser(context.Background(), test.existing))
			}
			// insert or update the user
			err := suite.mongoAdapter.UpdateUser(context.Background(), test.input)
			if test.expectedErr != nil {
				test.expectedErr(suite.T(), err)
			} else {
				suite.Require().NoError(err)
			}
			if test.expectedDB != nil {
				test.expectedDB(test.existing, test.input, suite.userCollection)
			}
		})
	}
}

func (suite *MongoDBTestSuite) TestListUsers() {
	tests := []struct {
		name          string
		existing      []*model.User
		query         ports.ListUsersQuery
		expectedErr   assert.ErrorAssertionFunc
		expectedUsers []model.User
	}{
		{
			name: "2 out 3 due to county filter",
			existing: []*model.User{
				{
					ID:           mustID(suite.T(), "3b3e9e2a13d54a68b5c58e6e").Hex(),
					FirstName:    "fn1",
					LastName:     "ln1",
					Nickname:     "n1",
					Email:        "e1",
					PasswordHash: "h1",
					Country:      "uk",
					CreatedAt:    dummyTime.Add(-10 * time.Minute),
					UpdatedAt:    dummyTime,
				},
				{
					ID:           mustID(suite.T(), "3b3e9e2a13d54a68b5c58e6f").Hex(),
					FirstName:    "fn2",
					LastName:     "ln2",
					Nickname:     "n2",
					Email:        "e2",
					PasswordHash: "h2",
					Country:      "br",
					CreatedAt:    dummyTime.Add(-5 * time.Minute),
					UpdatedAt:    dummyTime,
				},
				{
					ID:           mustID(suite.T(), "3b3e9e2a13d54a68b5c58e6a").Hex(),
					FirstName:    "fn3",
					LastName:     "ln3",
					Nickname:     "n3",
					Email:        "e3",
					PasswordHash: "h3",
					Country:      "us",
					CreatedAt:    dummyTime.Add(-1 * time.Minute),
					UpdatedAt:    dummyTime,
				},
			},
			query: ports.ListUsersQuery{
				Countries: []string{"uk", "br"},
			},
			expectedUsers: []model.User{
				{
					ID:           mustID(suite.T(), "3b3e9e2a13d54a68b5c58e6e").Hex(),
					FirstName:    "fn1",
					LastName:     "ln1",
					Nickname:     "n1",
					Email:        "e1",
					PasswordHash: "h1",
					Country:      "uk",
					CreatedAt:    dummyTime.Add(-10 * time.Minute),
					UpdatedAt:    dummyTime,
				},
				{
					ID:           mustID(suite.T(), "3b3e9e2a13d54a68b5c58e6f").Hex(),
					FirstName:    "fn2",
					LastName:     "ln2",
					Nickname:     "n2",
					Email:        "e2",
					PasswordHash: "h2",
					Country:      "br",
					CreatedAt:    dummyTime.Add(-5 * time.Minute),
					UpdatedAt:    dummyTime,
				},
			},
		},
		{
			name: "filtering 1 out of 3 on time-window",
			existing: []*model.User{
				{
					ID:           mustID(suite.T(), "3b3e9e2a13d54a68b5c58e6e").Hex(),
					FirstName:    "fn1",
					LastName:     "ln1",
					Nickname:     "n1",
					Email:        "e1",
					PasswordHash: "h1",
					Country:      "uk",
					CreatedAt:    dummyTime.Add(-10 * time.Minute),
					UpdatedAt:    dummyTime,
				},
				{
					ID:           mustID(suite.T(), "3b3e9e2a13d54a68b5c58e6f").Hex(),
					FirstName:    "fn2",
					LastName:     "ln2",
					Nickname:     "n2",
					Email:        "e2",
					PasswordHash: "h2",
					Country:      "br",
					CreatedAt:    dummyTime.Add(-5 * time.Minute),
					UpdatedAt:    dummyTime,
				},
				{
					ID:           mustID(suite.T(), "3b3e9e2a13d54a68b5c58e6a").Hex(),
					FirstName:    "fn3",
					LastName:     "ln3",
					Nickname:     "n3",
					Email:        "e3",
					PasswordHash: "h3",
					Country:      "us",
					CreatedAt:    dummyTime.Add(-1 * time.Minute),
					UpdatedAt:    dummyTime,
				},
			},
			query: ports.ListUsersQuery{
				CreatedAfter:  dummyTime.Add(-6 * time.Minute),
				CreatedBefore: dummyTime.Add(-4 * time.Minute),
			},
			expectedUsers: []model.User{
				{
					ID:           mustID(suite.T(), "3b3e9e2a13d54a68b5c58e6f").Hex(),
					FirstName:    "fn2",
					LastName:     "ln2",
					Nickname:     "n2",
					Email:        "e2",
					PasswordHash: "h2",
					Country:      "br",
					CreatedAt:    dummyTime.Add(-5 * time.Minute),
					UpdatedAt:    dummyTime,
				},
			},
		},
		{
			name: "pagination 2 first items",
			existing: []*model.User{
				{
					ID:           mustID(suite.T(), "3b3e9e2a13d54a68b5c58e6e").Hex(),
					FirstName:    "fn1",
					LastName:     "ln1",
					Nickname:     "n1",
					Email:        "e1",
					PasswordHash: "h1",
					Country:      "uk",
					CreatedAt:    dummyTime.Add(-10 * time.Minute),
					UpdatedAt:    dummyTime,
				},
				{
					ID:           mustID(suite.T(), "3b3e9e2a13d54a68b5c58e6f").Hex(),
					FirstName:    "fn2",
					LastName:     "ln2",
					Nickname:     "n2",
					Email:        "e2",
					PasswordHash: "h2",
					Country:      "br",
					CreatedAt:    dummyTime.Add(-5 * time.Minute),
					UpdatedAt:    dummyTime,
				},
				{
					ID:           mustID(suite.T(), "3b3e9e2a13d54a68b5c58e6a").Hex(),
					FirstName:    "fn3",
					LastName:     "ln3",
					Nickname:     "n3",
					Email:        "e3",
					PasswordHash: "h3",
					Country:      "us",
					CreatedAt:    dummyTime.Add(-1 * time.Minute),
					UpdatedAt:    dummyTime,
				},
			},
			query: ports.ListUsersQuery{
				Limit: 2,
			},
			expectedUsers: []model.User{
				{
					ID:           mustID(suite.T(), "3b3e9e2a13d54a68b5c58e6e").Hex(),
					FirstName:    "fn1",
					LastName:     "ln1",
					Nickname:     "n1",
					Email:        "e1",
					PasswordHash: "h1",
					Country:      "uk",
					CreatedAt:    dummyTime.Add(-10 * time.Minute),
					UpdatedAt:    dummyTime,
				},
				{
					ID:           mustID(suite.T(), "3b3e9e2a13d54a68b5c58e6f").Hex(),
					FirstName:    "fn2",
					LastName:     "ln2",
					Nickname:     "n2",
					Email:        "e2",
					PasswordHash: "h2",
					Country:      "br",
					CreatedAt:    dummyTime.Add(-5 * time.Minute),
					UpdatedAt:    dummyTime,
				},
			},
		},
		{
			name: "pagination 3rd item",
			existing: []*model.User{
				{
					ID:           mustID(suite.T(), "3b3e9e2a13d54a68b5c58e6e").Hex(),
					FirstName:    "fn1",
					LastName:     "ln1",
					Nickname:     "n1",
					Email:        "e1",
					PasswordHash: "h1",
					Country:      "uk",
					CreatedAt:    dummyTime.Add(-10 * time.Minute),
					UpdatedAt:    dummyTime,
				},
				{
					ID:           mustID(suite.T(), "3b3e9e2a13d54a68b5c58e6f").Hex(),
					FirstName:    "fn2",
					LastName:     "ln2",
					Nickname:     "n2",
					Email:        "e2",
					PasswordHash: "h2",
					Country:      "br",
					CreatedAt:    dummyTime.Add(-5 * time.Minute),
					UpdatedAt:    dummyTime,
				},
				{
					ID:           mustID(suite.T(), "3b3e9e2a13d54a68b5c58e6a").Hex(),
					FirstName:    "fn3",
					LastName:     "ln3",
					Nickname:     "n3",
					Email:        "e3",
					PasswordHash: "h3",
					Country:      "us",
					CreatedAt:    dummyTime.Add(-1 * time.Minute),
					UpdatedAt:    dummyTime,
				},
			},
			query: ports.ListUsersQuery{
				Limit:  2,
				Offset: 2,
			},
			expectedUsers: []model.User{
				{
					ID:           mustID(suite.T(), "3b3e9e2a13d54a68b5c58e6a").Hex(),
					FirstName:    "fn3",
					LastName:     "ln3",
					Nickname:     "n3",
					Email:        "e3",
					PasswordHash: "h3",
					Country:      "us",
					CreatedAt:    dummyTime.Add(-1 * time.Minute),
					UpdatedAt:    dummyTime,
				},
			},
		},
		{
			name: "offset out of range returns no item",
			existing: []*model.User{
				{
					ID:           mustID(suite.T(), "3b3e9e2a13d54a68b5c58e6e").Hex(),
					FirstName:    "fn1",
					LastName:     "ln1",
					Nickname:     "n1",
					Email:        "e1",
					PasswordHash: "h1",
					Country:      "uk",
					CreatedAt:    dummyTime.Add(-10 * time.Minute),
					UpdatedAt:    dummyTime,
				},
				{
					ID:           mustID(suite.T(), "3b3e9e2a13d54a68b5c58e6f").Hex(),
					FirstName:    "fn2",
					LastName:     "ln2",
					Nickname:     "n2",
					Email:        "e2",
					PasswordHash: "h2",
					Country:      "br",
					CreatedAt:    dummyTime.Add(-5 * time.Minute),
					UpdatedAt:    dummyTime,
				},
				{
					ID:           mustID(suite.T(), "3b3e9e2a13d54a68b5c58e6a").Hex(),
					FirstName:    "fn3",
					LastName:     "ln3",
					Nickname:     "n3",
					Email:        "e3",
					PasswordHash: "h3",
					Country:      "us",
					CreatedAt:    dummyTime.Add(-1 * time.Minute),
					UpdatedAt:    dummyTime,
				},
			},
			query: ports.ListUsersQuery{
				Limit:  2,
				Offset: 3,
			},
			expectedUsers: []model.User{},
		},
		{
			name: "1 out of 3 because 2 match the country filter and only 1 is not deleted",
			existing: []*model.User{
				{
					ID:           mustID(suite.T(), "3b3e9e2a13d54a68b5c58e6e").Hex(),
					FirstName:    "fn1",
					LastName:     "ln1",
					Nickname:     "n1",
					Email:        "e1",
					PasswordHash: "h1",
					Country:      "uk",
					CreatedAt:    dummyTime.Add(-10 * time.Minute),
					UpdatedAt:    dummyTime,
				},
				{
					ID:           mustID(suite.T(), "3b3e9e2a13d54a68b5c58e6f").Hex(),
					FirstName:    "fn2",
					LastName:     "ln2",
					Nickname:     "n2",
					Email:        "e2",
					PasswordHash: "h2",
					Country:      "br",
					CreatedAt:    dummyTime.Add(-5 * time.Minute),
					UpdatedAt:    dummyTime,
					DeletedAt:    dummyTime,
				},
				{
					ID:           mustID(suite.T(), "3b3e9e2a13d54a68b5c58e6a").Hex(),
					FirstName:    "fn3",
					LastName:     "ln3",
					Nickname:     "n3",
					Email:        "e3",
					PasswordHash: "h3",
					Country:      "us",
					CreatedAt:    dummyTime.Add(-1 * time.Minute),
					UpdatedAt:    dummyTime,
				},
			},
			query: ports.ListUsersQuery{
				Countries: []string{"uk", "br"},
			},
			expectedUsers: []model.User{
				{
					ID:           mustID(suite.T(), "3b3e9e2a13d54a68b5c58e6e").Hex(),
					FirstName:    "fn1",
					LastName:     "ln1",
					Nickname:     "n1",
					Email:        "e1",
					PasswordHash: "h1",
					Country:      "uk",
					CreatedAt:    dummyTime.Add(-10 * time.Minute),
					UpdatedAt:    dummyTime,
				},
			},
		},
	}

	for _, test := range tests {
		suite.Run(test.name, func() {

			_, err := suite.db.Database("faceittha").Collection("users").DeleteMany(context.Background(), bson.D{})
			suite.Require().NoError(err)

			if len(test.existing) > 0 {
				for _, u := range test.existing {
					suite.Require().NoError(suite.mongoAdapter.SaveUser(context.Background(), u))
				}
			}
			// insert or update the user
			res, err := suite.mongoAdapter.ListUsers(context.Background(), test.query)
			if test.expectedErr != nil {
				test.expectedErr(suite.T(), err)
			} else {
				suite.Require().NoError(err)
			}
			suite.Equal(test.expectedUsers, res.Users)

		})
	}
}

func (suite *MongoDBTestSuite) TestDeleteUser() {
	tests := []struct {
		name        string
		existing    *model.User
		query       ports.DeleteUserQuery
		expectedErr error
		expectedDB  func(collection *mongo.Collection)
	}{
		{
			name: "soft-delete preserve data for auditing",
			existing: &model.User{
				ID:           mustID(suite.T(), "3b3e9e2a13d54a68b5c58e6e").Hex(),
				FirstName:    "fn1",
				LastName:     "ln1",
				Nickname:     "n1",
				Email:        "e1",
				PasswordHash: "h1",
				Country:      "uk",
				CreatedAt:    dummyTime.Add(-10 * time.Minute),
				UpdatedAt:    dummyTime,
			},
			query: ports.DeleteUserQuery{
				ID:         mustID(suite.T(), "3b3e9e2a13d54a68b5c58e6e").Hex(),
				HardDelete: false,
			},
			expectedDB: func(collection *mongo.Collection) {
				objID := mustID(suite.T(), "3b3e9e2a13d54a68b5c58e6e")
				res := collection.FindOne(context.Background(), bson.D{bson.E{Key: "_id", Value: objID}})
				suite.NoError(res.Err())
				got := new(userDB)
				suite.Require().NoError(res.Decode(got))
				// deleted
				suite.NotZero(got.DeletedAt)
				expected := &userDB{
					ID:           mustID(suite.T(), "3b3e9e2a13d54a68b5c58e6e"),
					FirstName:    "fn1",
					LastName:     "ln1",
					Nickname:     "n1",
					Email:        "e1",
					PasswordHash: "h1",
					Country:      "uk",
					CreatedAt:    dummyTime.Add(-10 * time.Minute),
					UpdatedAt:    dummyTime,
					DeletedAt:    dummyTime,
				}
				suite.Equal(expected, got)
			},
		},
		{
			name: "hard-delete deletes the record",
			existing: &model.User{
				ID:           mustID(suite.T(), "3b3e9e2a13d54a68b5c58e6e").Hex(),
				FirstName:    "fn1",
				LastName:     "ln1",
				Nickname:     "n1",
				Email:        "e1",
				PasswordHash: "h1",
				Country:      "uk",
				CreatedAt:    dummyTime.Add(-10 * time.Minute),
				UpdatedAt:    dummyTime,
			},
			query: ports.DeleteUserQuery{
				ID:         mustID(suite.T(), "3b3e9e2a13d54a68b5c58e6e").Hex(),
				HardDelete: true,
			},
			expectedDB: func(collection *mongo.Collection) {
				objID := mustID(suite.T(), "3b3e9e2a13d54a68b5c58e6e")
				res := collection.FindOne(context.Background(), bson.D{bson.E{Key: "_id", Value: objID}})
				suite.ErrorIs(res.Err(), mongo.ErrNoDocuments)
			},
		},
		{
			name: "soft-delete non-existing record",
			query: ports.DeleteUserQuery{
				ID:         mustID(suite.T(), "3b3e9e2a13d54a68b5c58e6e").Hex(),
				HardDelete: false,
			},
		},
		{
			name: "hard-delete non-existing record",
			query: ports.DeleteUserQuery{
				ID:         mustID(suite.T(), "3b3e9e2a13d54a68b5c58e6e").Hex(),
				HardDelete: false,
			},
		},
	}

	for _, test := range tests {
		suite.Run(test.name, func() {

			_, err := suite.db.Database("faceittha").Collection("users").DeleteMany(context.Background(), bson.D{})
			suite.Require().NoError(err)
			if test.existing != nil {
				suite.Require().NoError(suite.mongoAdapter.SaveUser(context.Background(), test.existing))
			}

			// delete the user
			err = suite.mongoAdapter.DeleteUser(context.Background(), test.query)
			if test.expectedErr != nil {
				suite.ErrorIs(err, test.expectedErr)
			} else {
				suite.Require().NoError(err)
			}

			if test.expectedDB != nil {
				test.expectedDB(suite.userCollection)
			}

		})
	}
}

func TestMongoDBSuite(t *testing.T) {
	suite.Run(t, new(MongoDBTestSuite))
}

func mustID(t *testing.T, in string) primitive.ObjectID {
	objID, err := primitive.ObjectIDFromHex(in)
	require.NoError(t, err)
	return objID
}
