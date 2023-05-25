package ports

import (
	"context"

	"github.com/rbroggi/faceittha/internal/core/model"
)

// Sender is the port for publishing/informing/sending outbound user-events.
type Sender interface {
	// Send sends user-event data.
	Send(ctx context.Context, event model.UserEvent) error
}
