package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rbroggi/faceittha/internal/core/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockSender is a mock implementation of the Sender interface.
type MockSender struct {
	t *testing.T
	called bool
	UserEventAssertion func(t *testing.T, userEvent model.UserEvent)
	SendError     error
}

func (m *MockSender) Send(ctx context.Context, userEvent model.UserEvent) error {
	m.called = true
	if m.UserEventAssertion != nil {
		m.UserEventAssertion(m.t, userEvent)
	}
	return m.SendError
}

func TestInformer_Handle(t *testing.T) {
	sendingError := errors.New("sending error")
	tests := []struct {
		name             string
		userEvent        model.UserEvent
		userEventAssertion func(t *testing.T, userEvent model.UserEvent)
		sendError error
		callsSendMethod bool
		expectedError        func(t *testing.T, err error)
	
	}{
		{
			name: "update first name",
			userEvent: model.UserEvent{
				ID:     "1",
				Before: &model.User{
					FirstName: "name1",
				},
				After:  &model.User{
					FirstName: "name2",
				},
			},
			userEventAssertion: func(t *testing.T, userEvent model.UserEvent) {
				require.NotNil(t, userEvent.Before)
				require.NotNil(t, userEvent.After)
				require.Equal(t, "1", userEvent.ID)
				require.Equal(t, "name1", userEvent.Before.FirstName)
				require.Equal(t, "name2", userEvent.After.FirstName)
			},
			callsSendMethod: true,
		},
		{
			name: "user creation",
			userEvent: model.UserEvent{
				ID:     "1",
				After:  &model.User{
					FirstName: "name2",
				},
			},
			userEventAssertion: func(t *testing.T, userEvent model.UserEvent) {
				require.Nil(t, userEvent.Before)
				require.NotNil(t, userEvent.After)
				require.Equal(t, "1", userEvent.ID)
				require.Equal(t, "name2", userEvent.After.FirstName)
			},
			callsSendMethod: true,
		},
		{
			name: "user hard deletion",
			userEvent: model.UserEvent{
				ID:     "1",
				Before:  &model.User{
					FirstName: "name1",
				},
			},
			userEventAssertion: func(t *testing.T, userEvent model.UserEvent) {
				require.Nil(t, userEvent.After)
				require.NotNil(t, userEvent.Before)
				require.Equal(t, "1", userEvent.ID)
				require.Equal(t, "name1", userEvent.Before.FirstName)
			},
			callsSendMethod: true,
		},
		{
			name: "user soft deletion",
			userEvent: model.UserEvent{
				ID:     "1",
				Before:  &model.User{
					FirstName: "name1",
				},
				After:  &model.User{
					FirstName: "name1",
					DeletedAt: time.Now(),
				},
			},
			userEventAssertion: func(t *testing.T, userEvent model.UserEvent) {
				require.Nil(t, userEvent.After)
				require.NotNil(t, userEvent.Before)
				require.Equal(t, "1", userEvent.ID)
				require.Equal(t, "name1", userEvent.Before.FirstName)
			},
			callsSendMethod: true,
		},
		{
			name: "update only in the password hash should not send event",
			userEvent: model.UserEvent{
				ID:     "1",
				Before:  &model.User{
					FirstName: "name1",
					PasswordHash: "before",
				},
				After:  &model.User{
					FirstName: "name1",
					PasswordHash: "after",
				},
			},
			callsSendMethod: false,
		},
		{
			name: "error in sending event triggers error in handler",
			userEvent: model.UserEvent{ID: "1", Before: &model.User{FirstName: "name1"}, After: &model.User{FirstName: "name2"}},
			userEventAssertion: func(t *testing.T, userEvent model.UserEvent) {
				require.NotNil(t, userEvent.Before)
				require.NotNil(t, userEvent.After)
				require.Equal(t, "1", userEvent.ID)
				require.Equal(t, "name1", userEvent.Before.FirstName)
				require.Equal(t, "name2", userEvent.After.FirstName)
			},
			sendError:       sendingError,
			callsSendMethod: true,
			expectedError: func(t *testing.T, err error) {
				assert.ErrorIs(t, err, sendingError)
			},
		},

	}


	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sender := &MockSender{
				t: t,
				UserEventAssertion: test.userEventAssertion,
				SendError: test.sendError,
			}
			informer := NewInformer(sender)
			err := informer.Handle(context.Background(), test.userEvent)
			if test.expectedError != nil {
				test.expectedError(t, err)
			}
			require.Equal(t, test.callsSendMethod, sender.called)
		})
	}
}