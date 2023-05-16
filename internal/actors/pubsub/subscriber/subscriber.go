package subscriber

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/google/uuid"
	"github.com/rbroggi/faceittha/internal/core/model"
	"github.com/rbroggi/faceittha/internal/core/ports"

	log "github.com/sirupsen/logrus"
)

// SubscriberArgs contain the mandatory arguments to build a subscriber.
type SubscriberArgs struct {
	// Subscription is a pubsub subscription
	Subscription *pubsub.Subscription

	// UserEventHandler is a event handler
	UserEventHandler ports.UserEventHandler
}

// Subscriber is a pubsub async subscriber
type Subscriber struct {
	subscription     *pubsub.Subscription
	userEventHandler ports.UserEventHandler
}

// NewSubscriber creates a subscriber
func NewSubscriber(args SubscriberArgs) *Subscriber {
	return &Subscriber{
		subscription:     args.Subscription,
		userEventHandler: args.UserEventHandler,
	}
}

// Consume starts the subscriber. This is a blocking method and should be started in it's own go-routine.
// The way to terminate the method is to cancel the context in input.
func (s *Subscriber) Consume(ctx context.Context) error {
	if err := s.subscription.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {

		userEvent, err := decodeMsgIntoUserEvent(ctx, msg)
		if err != nil {
			log.WithError(err).Error("error decoding message into user-event")
			msg.Nack()
			return
		}

		if err := s.userEventHandler.Handle(ctx, *userEvent); err != nil {
			log.WithError(err).Error("error in user event handler")
			msg.Nack()
		} else {
			msg.Ack()
		}
	}); err != nil {
		return fmt.Errorf("error receiving messages from subscription: %w", err)
	}
	return nil
}

var (
	ErrIgnoreEvent = errors.New("event should be ignored")
)

func decodeMsgIntoUserEvent(ctx context.Context, msg *pubsub.Message) (*model.UserEvent, error) {
	if msg == nil {
		return nil, errors.New("cannot decode nil pubsub msg")
	}
	debeziumMsg := new(debeziumMessage)
	if err := json.Unmarshal(msg.Data, debeziumMsg); err != nil {
		return nil, fmt.Errorf("json unmarshal error: %w", err)
	}

	if debeziumMsg.Payload.Source.Table != "users" {
		return nil, ErrIgnoreEvent
	}

	userEvent := new(model.UserEvent)
	userEvent.ID = msg.ID
	userBefore, err := translateUserToModel(debeziumMsg.Payload.Before)
	if err != nil {
		return nil, ErrIgnoreEvent
	}
	userEvent.Before = userBefore
	userAfter, err := translateUserToModel(debeziumMsg.Payload.After)
	if err != nil {
		return nil, ErrIgnoreEvent
	}
	userEvent.After = userAfter

	return userEvent, nil
}

func translateUserToModel(dbzUser *debeziumUser) (*model.User, error) {
	if dbzUser == nil {
		return nil, nil
	}
	id, err := uuid.Parse(dbzUser.ID)
	if err != nil {
		return nil, err
	}

	deletedAt := time.Time{}
	if dbzUser.DeletedAt != nil {
		deletedAt = dbzUser.DeletedAt.Time
	}

	return &model.User{
		ID:           id,
		FirstName:    dbzUser.FirstName,
		LastName:     dbzUser.LastName,
		Nickname:     dbzUser.Nickname,
		Email:        dbzUser.Email,
		PasswordHash: dbzUser.PasswordHash,
		Country:      dbzUser.Country,
		CreatedAt:    dbzUser.CreatedAt.Time,
		UpdatedAt:    dbzUser.UpdatedAt.Time,
		DeletedAt:    deletedAt,
	}, nil
}

type debeziumMessage struct {
	// payload is the debezium segment containing the payload.
	Payload payload `json:"payload`
}

type payload struct {
	Op string `json:"op"`
	Source source `json:"source"`
	Before *debeziumUser `json:"before"`
	After  *debeziumUser `json:"after"`
}

type source struct {
	Schema string `json:"schema"`
	Table  string `json:"table"`
}

type debeziumUser struct {
	ID           string    `json:"id"`
	FirstName    string    `json:"first_name"`
	LastName     string    `json:"last_name"`
	Nickname     string    `json:"nickname"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"password_hash"`
	Country      string    `json:"country"`
	CreatedAt    UnixTime `json:"created_at"`
	UpdatedAt    UnixTime `json:"updated_at"`
	DeletedAt    *UnixTime `json:"deleted_at"`
}

// UnixTime is a custom type to allow us to redefine how to unmarshal from microseconds from epoch to time.Time
type UnixTime struct {
    time.Time
}

func (ut *UnixTime) UnmarshalJSON(b []byte) error {
    var timestamp int64
    err := json.Unmarshal(b, &timestamp)
    if err != nil {
        return err
    }
    ut.Time = time.Unix(0, timestamp*1000).UTC()
    return nil
}

func (ut UnixTime) MarshalJSON() ([]byte, error) {
    return []byte(strconv.FormatInt(ut.UnixNano()/1000, 10)), nil
}