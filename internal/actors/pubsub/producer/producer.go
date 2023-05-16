package producer

import (
	"context"
	"errors"
	"fmt"

	"cloud.google.com/go/pubsub"
	"github.com/rbroggi/faceittha/internal/core/model"
	v1 "github.com/rbroggi/faceittha/pkg/sdk/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// NewProducer creates a new producer.
func NewProducer(topic *pubsub.Topic) (*Producer, error) {
	if topic == nil {
		return nil, errors.New("topic is nil")
	}
	return &Producer{topic: topic}, nil
}
// Producer is the pubsub producer of user events.
type Producer struct {
	topic *pubsub.Topic
}

func (p *Producer) Send(ctx context.Context, event model.UserEvent) error {
	userEventProto := toProtoEvent(event)

	data, err := proto.Marshal(userEventProto)
	if err != nil {
		return fmt.Errorf("error marshaling user-event proto message: %w", err)
	}
	result := p.topic.Publish(ctx, &pubsub.Message{
		Data: data,
	})
	// Block until the result is returned and a server-generated
	// ID is returned for the published message.
	_, err = result.Get(ctx)
	if err != nil {
		return fmt.Errorf("pubsub: result.Get: %v", err)
	}
	return nil
}

func toProtoEvent(event model.UserEvent) *v1.UserEvent {
	return &v1.UserEvent{
		Before: toProtoUser(event.Before),
		After: toProtoUser(event.After),
	}
}

func toProtoUser(u *model.User) *v1.User {
	if u == nil {
		return nil
	}

	return &v1.User{
		Id:       u.ID.String(),
		FirstName: u.FirstName,
		LastName:  u.LastName,
		Nickname:  u.Nickname,
		Email:     u.Email,
		Country:   u.Country,
		CreatedAt: timestamppb.New(u.CreatedAt),
		UpdatedAt: timestamppb.New(u.UpdatedAt),
	}
}
