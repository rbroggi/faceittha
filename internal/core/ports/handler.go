package ports

import (
	"context"

	"github.com/rbroggi/faceittha/internal/core/model"
)

// UserEventHandler handles incoming UserEvents.
type UserEventHandler interface {
	// Handle will receive an incoming user event and handle it.
	Handle(ctx context.Context, userEvent model.UserEvent) error
}
