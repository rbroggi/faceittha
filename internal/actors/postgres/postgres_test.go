package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/google/uuid"
	"github.com/rbroggi/faceittha/internal/core/model"
	"github.com/rbroggi/faceittha/internal/core/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type PostgresDBTestSuite struct {
	suite.Suite
	db              *pg.DB
	postgresAdapter *PostgresDB
}

var (
	dummyTime = time.Now().Truncate(time.Second).UTC()
)

func (suite *PostgresDBTestSuite) SetupSuite() {
	url := os.Getenv("POSTGRESQL_URL")
	if url == "" {
		url = "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
	}
	opts, err := pg.ParseURL(url)
	suite.Require().NoError(err)
	db := pg.Connect(opts)
	suite.Require().NoError(db.Ping(context.Background()))
	dummyTimeFunc := func() time.Time {
		return dummyTime
	}
	pgDB, err := NewPostgresDB(PostgresDBArgs{DB: db}, WithNowFunc(dummyTimeFunc))
	suite.Require().NoError(err)
	suite.postgresAdapter = pgDB
	suite.db = db
}

func (suite *PostgresDBTestSuite) SetupTest() {
	_, err := suite.db.Exec("TRUNCATE TABLE faceittha.users")
	suite.Require().NoError(err)
}

func (suite *PostgresDBTestSuite) TearDownSuite() {
	// close the database connection after each test
	suite.Require().NoError(suite.db.Close())
}

func (suite *PostgresDBTestSuite) TestSaveUser() {
	tests := []struct {
		name        string
		input       *model.User
		expectedErr assert.ErrorAssertionFunc
		expectedDB  func(input *model.User, db *pg.DB)
	}{
		{
			name: "insert new user",
			input: &model.User{
				ID:           uuid.New(),
				FirstName:    "Jane",
				LastName:     "Doe",
				Nickname:     "jd",
				Email:        "newuser@example.com",
				PasswordHash: "hash",
				Country:      "UK",
			},
			expectedDB: func(input *model.User, db *pg.DB) {
				got := new(userDB)
				suite.NoError(db.Model(got).Where("id = ?", input.ID).Select())
				suite.Equal(got.ID, input.ID)
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
			err := suite.postgresAdapter.SaveUser(context.Background(), test.input)
			if test.expectedErr != nil {
				test.expectedErr(suite.T(), err)
			} else {
				suite.Require().NoError(err)
			}
			if test.expectedDB != nil {
				test.expectedDB(test.input, suite.db)
			}
		})
	}
}

func (suite *PostgresDBTestSuite) TestUpdateUser() {
	tests := []struct {
		name        string
		existing    *model.User
		input       *model.User
		expectedErr assert.ErrorAssertionFunc
		expectedDB  func(existing, input *model.User, db *pg.DB)
	}{
		{
			name: "save existing user (update)",
			existing: &model.User{
				ID:           uuid.MustParse("3b3e9e2a-13d5-4a68-b5c5-8e60a5b5d5de"),
				FirstName:    "Jane",
				LastName:     "Doe",
				Nickname:     "jd",
				Email:        "newuser@example.com",
				PasswordHash: "hash",
				Country:      "UK",
			},
			input: &model.User{
				ID: uuid.MustParse("3b3e9e2a-13d5-4a68-b5c5-8e60a5b5d5de"),
				FirstName:    "Jane2",
				LastName:     "Doe2",
				Nickname:     "jd2",
				Email:        "newuser2@example.com",
				PasswordHash: "hash2",
				Country:      "BR",
			},
			expectedDB: func(existing, input *model.User, db *pg.DB) {
				got := new(userDB)
				suite.NoError(db.Model(got).Where("id = ?", input.ID).Select())
				suite.Equal(got.ID, existing.ID)
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
				suite.Require().NoError(suite.postgresAdapter.SaveUser(context.Background(), test.existing))
			}
			// insert or update the user
			err := suite.postgresAdapter.UpdateUser(context.Background(), test.input)
			if test.expectedErr != nil {
				test.expectedErr(suite.T(), err)
			} else {
				suite.Require().NoError(err)
			}
			if test.expectedDB != nil {
				test.expectedDB(test.existing, test.input, suite.db)
			}
		})
	}
}

func (suite *PostgresDBTestSuite) TestListUsers() {
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
					ID:           uuid.MustParse("3b3e9e2a-13d5-4a68-b5c5-8e60a5b5d5de"),
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
					ID:           uuid.MustParse("3b3e9e2a-13d5-4a68-b5c5-8e60a5b5d5df"),
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
					ID:           uuid.MustParse("3b3e9e2a-13d5-4a68-b5c5-8e60a5b5d5da"),
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
					ID:           uuid.MustParse("3b3e9e2a-13d5-4a68-b5c5-8e60a5b5d5de"),
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
					ID:           uuid.MustParse("3b3e9e2a-13d5-4a68-b5c5-8e60a5b5d5df"),
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
					ID:           uuid.MustParse("3b3e9e2a-13d5-4a68-b5c5-8e60a5b5d5de"),
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
					ID:           uuid.MustParse("3b3e9e2a-13d5-4a68-b5c5-8e60a5b5d5df"),
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
					ID:           uuid.MustParse("3b3e9e2a-13d5-4a68-b5c5-8e60a5b5d5da"),
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
					ID:           uuid.MustParse("3b3e9e2a-13d5-4a68-b5c5-8e60a5b5d5df"),
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
					ID:           uuid.MustParse("3b3e9e2a-13d5-4a68-b5c5-8e60a5b5d5de"),
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
					ID:           uuid.MustParse("3b3e9e2a-13d5-4a68-b5c5-8e60a5b5d5df"),
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
					ID:           uuid.MustParse("3b3e9e2a-13d5-4a68-b5c5-8e60a5b5d5da"),
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
					ID:           uuid.MustParse("3b3e9e2a-13d5-4a68-b5c5-8e60a5b5d5de"),
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
					ID:           uuid.MustParse("3b3e9e2a-13d5-4a68-b5c5-8e60a5b5d5df"),
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
					ID:           uuid.MustParse("3b3e9e2a-13d5-4a68-b5c5-8e60a5b5d5de"),
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
					ID:           uuid.MustParse("3b3e9e2a-13d5-4a68-b5c5-8e60a5b5d5df"),
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
					ID:           uuid.MustParse("3b3e9e2a-13d5-4a68-b5c5-8e60a5b5d5da"),
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
					ID:           uuid.MustParse("3b3e9e2a-13d5-4a68-b5c5-8e60a5b5d5da"),
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
					ID:           uuid.MustParse("3b3e9e2a-13d5-4a68-b5c5-8e60a5b5d5de"),
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
					ID:           uuid.MustParse("3b3e9e2a-13d5-4a68-b5c5-8e60a5b5d5df"),
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
					ID:           uuid.MustParse("3b3e9e2a-13d5-4a68-b5c5-8e60a5b5d5da"),
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
					ID:           uuid.MustParse("3b3e9e2a-13d5-4a68-b5c5-8e60a5b5d5de"),
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
					ID:           uuid.MustParse("3b3e9e2a-13d5-4a68-b5c5-8e60a5b5d5df"),
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
					ID:           uuid.MustParse("3b3e9e2a-13d5-4a68-b5c5-8e60a5b5d5da"),
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
					ID:           uuid.MustParse("3b3e9e2a-13d5-4a68-b5c5-8e60a5b5d5de"),
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

			_, err := suite.db.Exec("TRUNCATE TABLE faceittha.users")
			suite.Require().NoError(err)

			if len(test.existing) > 0 {
				for _, u := range test.existing {
					suite.Require().NoError(suite.postgresAdapter.SaveUser(context.Background(), u))
				}
			}
			// insert or update the user
			res, err := suite.postgresAdapter.ListUsers(context.Background(), test.query)
			if test.expectedErr != nil {
				test.expectedErr(suite.T(), err)
			} else {
				suite.Require().NoError(err)
			}
			suite.Equal(test.expectedUsers, res.Users)

		})
	}
}

func (suite *PostgresDBTestSuite) TestDeleteUser() {
	tests := []struct {
		name        string
		existing    *model.User
		query       ports.DeleteUserQuery
		expectedErr error
		expectedDB  func(db *pg.DB)
	}{
		{
			name: "soft-delete preserve data for auditing",
			existing: &model.User{
				ID:           uuid.MustParse("3b3e9e2a-13d5-4a68-b5c5-8e60a5b5d5de"),
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
				ID:         uuid.MustParse("3b3e9e2a-13d5-4a68-b5c5-8e60a5b5d5de"),
				HardDelete: false,
			},
			expectedDB: func(db *pg.DB) {
				got := new(userDB)
				suite.NoError(suite.db.Model(got).Where("id = ?", uuid.MustParse("3b3e9e2a-13d5-4a68-b5c5-8e60a5b5d5de")).Select())
				// deleted
				suite.NotZero(got.DeletedAt)
				expected := &userDB{
					ID:           uuid.MustParse("3b3e9e2a-13d5-4a68-b5c5-8e60a5b5d5de"),
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
				ID:           uuid.MustParse("3b3e9e2a-13d5-4a68-b5c5-8e60a5b5d5de"),
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
				ID:         uuid.MustParse("3b3e9e2a-13d5-4a68-b5c5-8e60a5b5d5de"),
				HardDelete: true,
			},
			expectedDB: func(db *pg.DB) {
				got := new(userDB)
				err := suite.db.Model(got).Where("id = ?", uuid.MustParse("3b3e9e2a-13d5-4a68-b5c5-8e60a5b5d5de")).Select()
				suite.ErrorIs(err, pg.ErrNoRows)
			},
		},
		{
			name: "soft-delete non-existing record",
			query: ports.DeleteUserQuery{
				ID:         uuid.MustParse("3b3e9e2a-13d5-4a68-b5c5-5e60a5b5d5de"),
				HardDelete: false,
			},
		},
		{
			name: "hard-delete non-existing record",
			query: ports.DeleteUserQuery{
				ID:         uuid.MustParse("3b3e9e2a-13d5-4a68-b5c5-6e60a5b5d5de"),
				HardDelete: false,
			},
		},
	}

	for _, test := range tests {
		suite.Run(test.name, func() {

			_, err := suite.db.Exec("TRUNCATE TABLE faceittha.users")
			suite.Require().NoError(err)
			if test.existing != nil {
				suite.Require().NoError(suite.postgresAdapter.SaveUser(context.Background(), test.existing))
			}

			// delete the user
			err = suite.postgresAdapter.DeleteUser(context.Background(), test.query)
			if test.expectedErr != nil {
				suite.ErrorIs(err, test.expectedErr)
			} else {
				suite.Require().NoError(err)
			}

			if test.expectedDB != nil {
				test.expectedDB(suite.db)
			}

		})
	}
}

func TestPostgresDBSuite(t *testing.T) {
	suite.Run(t, new(PostgresDBTestSuite))
}
