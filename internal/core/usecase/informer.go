package usecase

import (
	"context"
	"fmt"

	"github.com/rbroggi/faceittha/internal/core/model"
	"github.com/rbroggi/faceittha/internal/core/ports"
)

// NewInformer builds a new informer.
func NewInformer(sender ports.Sender) *Informer {
	return &Informer{sender: sender}
}

// Informer adapts CDC events to a public-facing event. It publicly 'informs' about user changes.
type Informer struct {
	sender ports.Sender
}


func (i *Informer) Handle(ctx context.Context, userEvent model.UserEvent) error {

	// 1. we don't want to publish changes in password
	if userEvent.Before != nil {
		userEvent.Before.PasswordHash = ""
	}
	if userEvent.After != nil {
		userEvent.After.PasswordHash = ""
	}

	// 2. we don't want to publish soft vs hard deletions. This is internal complexities of this service and 
	// does not make sense to share this info with consumers.
	if userEvent.Before != nil && userEvent.After != nil && !userEvent.After.DeletedAt.IsZero() {
		userEvent.After = nil
	}

	// this happens if there were only changes in password hash
	if eventsAreEqual(userEvent.Before, userEvent.After) {
		return nil
	}

	if err := i.sender.Send(ctx, userEvent); err != nil {
		return fmt.Errorf("error sending user event ID [%s]: %w", userEvent.ID, err)
	}

	return nil
}

func eventsAreEqual(before *model.User, after *model.User) bool {
	if before == nil && after == nil {
		return true
	}
	if before == nil && after != nil {
		return false
	}
	if before != nil && after == nil {
		return false
	}
	return *before == *after
}