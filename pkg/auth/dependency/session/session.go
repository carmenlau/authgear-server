package session

import (
	"time"

	"github.com/skygeario/skygear-server/pkg/core/authn"
)

type Attrs struct {
	UserID string `json:"user_id"`

	PrincipalID        string              `json:"principal_id"`
	PrincipalType      authn.PrincipalType `json:"principal_type"`
	PrincipalUpdatedAt time.Time           `json:"principal_updated_at"`

	AuthenticatorID         string                        `json:"authenticator_id,omitempty"`
	AuthenticatorType       authn.AuthenticatorType       `json:"authenticator_type,omitempty"`
	AuthenticatorOOBChannel authn.AuthenticatorOOBChannel `json:"authenticator_oob_channel,omitempty"`
	AuthenticatorUpdatedAt  *time.Time                    `json:"authenticator_updated_at,omitempty"`
}

type Session struct {
	ID    string `json:"id"`
	AppID string `json:"app_id"`

	Attrs Attrs `json:"attrs"`

	InitialAccess AccessEvent `json:"initial_access"`
	LastAccess    AccessEvent `json:"last_access"`

	CreatedAt  time.Time `json:"created_at"`
	AccessedAt time.Time `json:"accessed_at"`

	TokenHash string `json:"token_hash"`
}

func (s *Session) SessionAttrs() *Attrs {
	return &s.Attrs
}

type CreateReason string

const (
	CreateReasonSignup = "signup"
	CreateReasonLogin  = "login"
)