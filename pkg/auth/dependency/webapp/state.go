package webapp

import (
	"errors"

	"github.com/authgear/authgear-server/pkg/auth/dependency/interaction"
	corerand "github.com/authgear/authgear-server/pkg/core/rand"
	"github.com/authgear/authgear-server/pkg/core/skyerr"
)

var ErrStateNotFound = errors.New("state not found")

var (
	stateIDAlphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	stateIDLength   = 32
)

const (
	ExtraSSOAction      string = "sso_action"
	ExtraSSONonce       string = "sso_nonce"
	ExtraSSORedirectURI string = "sso_redirect_uri"
	ExtraUserID         string = "user_id"
)

// State management
// The webapp adopts the Post/Redirect/Get pattern.
//
// In this pattern, we cannot persist state directly by rendering
// hidden form fields in the response of POST request.
//
// Here we use a simple approach to work around this limitation.
//
// In the first POST request of a flow, a state object is created.
// The x_sid query parameter in the URL identities a state object.
//
// This approach does not use cookie at all.
type State struct {
	// ID is a cryptographically random string.
	ID string `json:"id"`

	// Interaction is the interaction associated with this state.
	Interaction *interaction.Interaction `json:"interaction"`

	// FIXME(webapp): Clear error correctly.
	// Error is either reset to nil or set to non-nil in every POST request.
	Error *skyerr.APIError `json:"error"`

	// FIXME(webapp): Move AnonymousUserID to somewhere else.
	// AnonymousUserID is the ID of anonymous user during promotion flow.
	AnonymousUserID string `json:"anonymous_user_id,omitempty"`

	// FIXME(webapp): Unify with Interaction.Extra.
	// Extra is used to persist extra data across the interaction.
	Extra map[string]interface{} `json:"extra,omitempty"`
}

func NewState() *State {
	return &State{
		ID:    corerand.StringWithAlphabet(stateIDLength, stateIDAlphabet, corerand.SecureRand),
		Extra: make(map[string]interface{}),
	}
}

func (s *State) SetError(err error) {
	s.Error = skyerr.AsAPIError(err)
}
