package nonblocking

import (
	"fmt"

	"github.com/authgear/authgear-server/pkg/api/event"
	"github.com/authgear/authgear-server/pkg/api/model"
)

const (
	IdentityLoginIDUpdatedFormat string = "identity.%s.updated"
)

type IdentityLoginIDUpdatedEvent struct {
	User        model.User     `json:"user"`
	NewIdentity model.Identity `json:"new_identity"`
	OldIdentity model.Identity `json:"old_identity"`
	LoginIDType string         `json:"-"`
	AdminAPI    bool           `json:"-"`
}

func NewIdentityLoginIDUpdatedEvent(
	user model.User,
	newIdentity model.Identity,
	oldIdentity model.Identity,
	loginIDType string,
	adminAPI bool,
) *IdentityLoginIDUpdatedEvent {
	if checkIdentityEventSupportLoginIDType(loginIDType) {
		return &IdentityLoginIDUpdatedEvent{
			User:        user,
			NewIdentity: newIdentity,
			OldIdentity: oldIdentity,
			LoginIDType: loginIDType,
			AdminAPI:    adminAPI,
		}
	}
	return nil
}

func (e *IdentityLoginIDUpdatedEvent) NonBlockingEventType() event.Type {
	return event.Type(fmt.Sprintf(IdentityLoginIDUpdatedFormat, e.LoginIDType))
}

func (e *IdentityLoginIDUpdatedEvent) UserID() string {
	return e.User.ID
}

func (e *IdentityLoginIDUpdatedEvent) IsAdminAPI() bool {
	return e.AdminAPI
}

var _ event.NonBlockingPayload = &IdentityLoginIDUpdatedEvent{}