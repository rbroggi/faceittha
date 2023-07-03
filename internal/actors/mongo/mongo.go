package mongo

import (
	"context"
	"errors"
	"time"

	"github.com/rbroggi/faceittha/internal/core/model"
	"github.com/rbroggi/faceittha/internal/core/ports"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoDB is a postgress adapter for persistance.
type MongoDB struct {
	userCollection *mongo.Collection
	nowFunc        func() time.Time
}

// MongoDBArgs are the mandatory arguments for the creation of a MongoDB
type MongoDBArgs struct {
	// UserCollection is a mongo collection
	UserCollection *mongo.Collection
}

// MongoDBOptArgs are the optional arguments for building a MongoDB
type MongoDBOptArgs = func(*MongoDB)

// WithNowFunc can be used to override the nowFunc. Useful for testing.
func WithNowFunc(nowFunc func() time.Time) MongoDBOptArgs {
	return func(p *MongoDB) {
		p.nowFunc = nowFunc
	}
}

// NewMongoDB creates a new MongoDB.
func NewMongoDB(args MongoDBArgs, optArgs ...MongoDBOptArgs) (*MongoDB, error) {
	pg := &MongoDB{userCollection: args.UserCollection, nowFunc: func() time.Time { return time.Now().UTC() }}
	for _, opt := range optArgs {
		opt(pg)
	}
	return pg, nil
}

// SaveUser will save the user in the database.
func (p *MongoDB) SaveUser(ctx context.Context, user *model.User) error {

	if user == nil {
		return errors.New("nil user passed to save method")
	}

	dbUser := p.toDBModel(user)
	if _, err := p.userCollection.InsertOne(ctx, dbUser); err != nil {
		return err
	}

	user.ID = dbUser.ID.Hex()
	user.CreatedAt = dbUser.CreatedAt
	user.UpdatedAt = dbUser.UpdatedAt
	return nil
}

// UpdateUser will update user. It returns model.ErrNotFound if the input user does not exist.
func (p *MongoDB) UpdateUser(ctx context.Context, user *model.User) error {
	if user == nil {
		return errors.New("nil user passed to update method")
	}

	objectID, err := primitive.ObjectIDFromHex(user.ID)

	toUpdate := p.updateExisting(user)
	res, err := p.userCollection.UpdateByID(ctx, objectID, toUpdate)
	if err != nil {
		return err
	}
	if res.ModifiedCount < 1 {
		return model.ErrNotFound
	}

	existingUser := new(userDB)
	if err := p.userCollection.FindOne(ctx, bson.D{{"_id", objectID}}).Decode(existingUser); err != nil {
		return err
	}
	user.FirstName = existingUser.FirstName
	user.LastName = existingUser.LastName
	user.Nickname = existingUser.Nickname
	user.Email = existingUser.Email
	user.PasswordHash = existingUser.PasswordHash
	user.Country = existingUser.Country
	user.CreatedAt = existingUser.CreatedAt
	user.UpdatedAt = existingUser.UpdatedAt
	return nil
}

// ListUsers list users matching the parameters in input
func (p *MongoDB) ListUsers(ctx context.Context, query ports.ListUsersQuery) (*ports.ListUsersResult, error) {

	filters := bson.M{}
	filters["deleted_at"] = bson.M{"$exists": false}
	if len(query.Countries) > 0 {
		filters["country"] = bson.M{"$in": query.Countries}
	}
	timeFilter := bson.M{}
	if !query.CreatedAfter.IsZero() {
		timeFilter["$gte"] = primitive.NewDateTimeFromTime(query.CreatedAfter)
	}
	if !query.CreatedBefore.IsZero() {
		timeFilter["$lte"] = primitive.NewDateTimeFromTime(query.CreatedBefore)
	}
	if len(timeFilter) > 0 {
		filters["created_at"] = timeFilter
	}

	opts := new(options.FindOptions)
	if query.Limit != uint32(0) {
		l := int64(query.Limit)
		opts.Limit = &l
	}
	if query.Offset != uint32(0) {
		s := int64(query.Offset)
		opts.Skip = &s
	}
	opts = opts.SetSort(bson.D{{"created_at", 1}})
	var users []userDB
	cursor, err := p.userCollection.Find(ctx, filters, opts)
	if err != nil {
		return nil, err
	}
	if err := cursor.All(ctx, &users); err != nil {
		return nil, err
	}
	model := translateDBToModels(users)
	return &ports.ListUsersResult{
		Users: model,
	}, nil
}

// DeleteUser will delete a user from the database.
func (p *MongoDB) DeleteUser(ctx context.Context, query ports.DeleteUserQuery) error {
	objectID, err := primitive.ObjectIDFromHex(query.ID)
	if err != nil {
		return err
	}
	if query.HardDelete {
		if _, err := p.userCollection.DeleteOne(ctx, bson.D{{"_id", objectID}}); err != nil {
			return err
		}
		return nil
	}
	update := bson.D{{"$set", bson.D{{"deleted_at", p.nowFunc()}}}}
	if _, err := p.userCollection.UpdateByID(ctx, objectID, update); err != nil {
		return err
	}
	return nil
}

func (p *MongoDB) toDBModel(user *model.User) *userDB {
	dbUser := new(userDB)
	if len(user.ID) == 0 {
		dbUser.ID = primitive.NewObjectID()
		user.ID = dbUser.ID.Hex()
	} else {
		var err error
		dbUser.ID, err = primitive.ObjectIDFromHex(user.ID)
		if err != nil {
			return nil
		}
	}
	if user.FirstName != "" {
		dbUser.FirstName = user.FirstName
	}
	if user.LastName != "" {
		dbUser.LastName = user.LastName
	}
	if user.Nickname != "" {
		dbUser.Nickname = user.Nickname
	}
	if user.Email != "" {
		dbUser.Email = user.Email
	}
	if len(user.PasswordHash) != 0 {
		dbUser.PasswordHash = user.PasswordHash
	}
	if user.Country != "" {
		dbUser.Country = user.Country
	}
	if !user.CreatedAt.IsZero() {
		dbUser.CreatedAt = user.CreatedAt
	} else {
		dbUser.CreatedAt = p.nowFunc()
	}
	if !user.DeletedAt.IsZero() {
		dbUser.DeletedAt = user.CreatedAt
	}
	dbUser.UpdatedAt = p.nowFunc()
	return dbUser
}

func (p *MongoDB) updateExisting(user *model.User) bson.D {
	toUpdate := bson.D{}
	if user.FirstName != "" {
		toUpdate = append(toUpdate, bson.E{Key: "first_name", Value: user.FirstName})
	}
	if user.LastName != "" {
		toUpdate = append(toUpdate, bson.E{Key: "last_name", Value: user.LastName})
	}
	if user.Nickname != "" {
		toUpdate = append(toUpdate, bson.E{Key: "nickname", Value: user.Nickname})
	}
	if user.Email != "" {
		toUpdate = append(toUpdate, bson.E{Key: "email", Value: user.Email})
	}
	if len(user.PasswordHash) != 0 {
		toUpdate = append(toUpdate, bson.E{Key: "password_hash", Value: user.PasswordHash})
	}
	if user.Country != "" {
		toUpdate = append(toUpdate, bson.E{Key: "country", Value: user.Country})
	}
	if !user.DeletedAt.IsZero() {
		toUpdate = append(toUpdate, bson.E{Key: "deleted_at", Value: user.DeletedAt})
	}
	toUpdate = append(toUpdate, bson.E{Key: "updated_at", Value: user.UpdatedAt})

	return bson.D{{"$set",
		toUpdate,
	}}
}

func translateDBToModels(dbUsers []userDB) []model.User {
	models := make([]model.User, len(dbUsers))
	for i, dbUser := range dbUsers {
		models[i] = translateDBToModel(dbUser)
	}
	return models
}

func translateDBToModel(dbUser userDB) model.User {
	return model.User{
		ID:           dbUser.ID.Hex(),
		FirstName:    dbUser.FirstName,
		LastName:     dbUser.LastName,
		Nickname:     dbUser.Nickname,
		Email:        dbUser.Email,
		PasswordHash: dbUser.PasswordHash,
		Country:      dbUser.Country,
		CreatedAt:    dbUser.CreatedAt,
		UpdatedAt:    dbUser.UpdatedAt,
		DeletedAt:    dbUser.DeletedAt,
	}
}

type userDB struct {
	// ID unique identifier of the user.
	ID primitive.ObjectID `bson:"_id"`

	// FirstName is the user first name.
	FirstName string `bson:"first_name"`

	// LastName is the user last name.
	LastName string `bson:"last_name"`

	// Nickname is the user nickname
	Nickname string `bson:"nickname"`

	// Email is the user email
	Email string `bson:"email"`

	// PasswordHash contains the password hash.
	PasswordHash string `bson:"password_hash"`

	// Country is the user country
	Country string `bson:"country"`

	// CreatedAt is the time at which the user was created in the system.
	CreatedAt time.Time `bson:"created_at"`

	// UpdatedAt is the time at which the user was last updated
	UpdatedAt time.Time `bson:"updated_at"`

	// DeletedAt is the time at which the user was deleted. Zero-valued if user not deleted
	DeletedAt time.Time `bson:"deleted_at,omitempty"`
}
