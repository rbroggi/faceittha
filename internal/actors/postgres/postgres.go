package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/google/uuid"
	"github.com/rbroggi/faceittha/internal/core/model"
	"github.com/rbroggi/faceittha/internal/core/ports"
)

// PostgresDB is a postgress adapter for persistance.
type PostgresDB struct {
	db      *pg.DB
	nowFunc func() time.Time
}

// PostgresDBArgs are the mandatory arguments for the creation of a PostgresDB
type PostgresDBArgs struct {
	// DB is a postgres database handle
	DB *pg.DB
}

// PostgresDBOptArgs are the optional arguments for building a PostgresDB
type PostgresDBOptArgs = func(*PostgresDB)

// WithNowFunc can be used to override the nowFunc. Useful for testing.
func WithNowFunc(nowFunc func() time.Time) PostgresDBOptArgs {
	return func(p *PostgresDB) {
		p.nowFunc = nowFunc
	}
}

// NewPostgresDB creates a new PostgresDB.
func NewPostgresDB(args PostgresDBArgs, optArgs ...PostgresDBOptArgs) (*PostgresDB, error) {
	pg := &PostgresDB{db: args.DB, nowFunc: func() time.Time { return time.Now().UTC() }}
	for _, opt := range optArgs {
		opt(pg)
	}
	return pg, nil
}

// SaveUser will save the user in the database.
func (p *PostgresDB) SaveUser(ctx context.Context, user *model.User) error {

	if user == nil {
		return errors.New("nil user passed to save method")
	}

	existingUser := p.toDBModel(user)
	if _, err := p.db.Model(existingUser).Insert(); err != nil {
		return err
	}

	user.ID = existingUser.ID
	user.CreatedAt = existingUser.CreatedAt
	user.UpdatedAt = existingUser.UpdatedAt
	return nil
}

// UpdateUser will update user. It returns model.ErrNotFound if the input user does not exist.
func (p *PostgresDB) UpdateUser(ctx context.Context, user *model.User) error {
	if user == nil {
		return errors.New("nil user passed to update method")
	}

	conn := p.db.Conn()
	defer conn.Close()

	tx, err := conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	existingUser := new(userDB)
	err = tx.Model(existingUser).Where("id = ?", user.ID).Select()
	if err != nil && err != pg.ErrNoRows {
		return err
	} else if err == pg.ErrNoRows {
		return model.ErrNotFound
	}

	p.updateExisting(existingUser, user)
	if _, err := tx.Model(existingUser).WherePK().Update(); err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
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
func (p *PostgresDB) ListUsers(ctx context.Context, query ports.ListUsersQuery) (*ports.ListUsersResult, error) {
	var users []userDB
	q := p.db.Model(&users).Order("created_at ASC").Where("deleted_at IS NULL")

	if len(query.Countries) > 0 {
		q = q.WhereIn("country IN (?)", query.Countries)
	}
	if !query.CreatedAfter.IsZero() {
		q = q.Where("created_at > ?", query.CreatedAfter)
	}
	if !query.CreatedBefore.IsZero() {
		q = q.Where("created_at < ?", query.CreatedBefore)
	}
	if query.Limit != uint32(0) {
		q = q.Limit(int(query.Limit))
	}
	if query.Offset != uint32(0) {
		q = q.Offset(int(query.Offset))
	}
	if err := q.Select(); err != nil && err != pg.ErrNoRows {
		return nil, err
	}

	model := translateDBToModels(users)
	return &ports.ListUsersResult{
		Users: model,
	}, nil
}

// DeleteUser will delete a user from the database.
func (p *PostgresDB) DeleteUser(ctx context.Context, query ports.DeleteUserQuery) error {
	userDB := &userDB{ID: query.ID}
	if query.HardDelete {
		if _, err := p.db.Model(userDB).WherePK().Delete(); err != nil {
			return err
		}
		return nil
	}
	if _, err := p.db.Model(userDB).WherePK().Set("deleted_at = ?", p.nowFunc()).Update(); err != nil {
		return err
	}
	return nil
}

func (p *PostgresDB) toDBModel(user *model.User) *userDB {
	dbUser := new(userDB)
	if user.ID.String() == "" {
		user.ID = uuid.New()
	}
	dbUser.ID = user.ID
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

func (p *PostgresDB) updateExisting(existingDBUser *userDB, user *model.User) {
	if user.FirstName != "" {
		existingDBUser.FirstName = user.FirstName
	}
	if user.LastName != "" {
		existingDBUser.LastName = user.LastName
	}
	if user.Nickname != "" {
		existingDBUser.Nickname = user.Nickname
	}
	if user.Email != "" {
		existingDBUser.Email = user.Email
	}
	if len(user.PasswordHash) != 0 {
		existingDBUser.PasswordHash = user.PasswordHash
	}
	if user.Country != "" {
		existingDBUser.Country = user.Country
	}
	if !user.DeletedAt.IsZero() {
		existingDBUser.DeletedAt = user.CreatedAt
	}
	existingDBUser.UpdatedAt = p.nowFunc()
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
		ID:           dbUser.ID,
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
	tableName struct{} `pg:"faceittha.users"`

	// ID unique identifier of the user.
	ID uuid.UUID `pg:"id,type:uuid,default:uuid_generate_v4()"`

	// FirstName is the user first name.
	FirstName string `pg:"first_name"`

	// LastName is the user last name.
	LastName string `pg:"last_name"`

	// Nickname is the user nickname
	Nickname string `pg:"nickname"`

	// Email is the user email
	Email string `pg:"email"`

	// PasswordHash contains the password hash.
	PasswordHash string `pg:"password_hash"`

	// Country is the user country
	Country string `pg:"country"`

	// CreatedAt is the time at which the user was created in the system.
	CreatedAt time.Time `pg:"created_at"`

	// UpdatedAt is the time at which the user was last updated
	UpdatedAt time.Time `pg:"updated_at"`

	// DeletedAt is the time at which the user was deleted. Zero-valued if user not deleted
	DeletedAt time.Time `pg:"deleted_at"`
}
