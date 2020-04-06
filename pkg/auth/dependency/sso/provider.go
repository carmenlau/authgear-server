package sso

import (
	"github.com/skygeario/skygear-server/pkg/core/config"
)

type Provider interface {
	EncodeState(state State) (encodedState string, err error)
	DecodeState(encodedState string) (*State, error)

	EncodeSkygearAuthorizationCode(SkygearAuthorizationCode) (code string, err error)
	DecodeSkygearAuthorizationCode(code string) (*SkygearAuthorizationCode, error)

	IsValidCallbackURL(config.OAuthClientConfiguration, string) bool

	IsExternalAccessTokenFlowEnabled() bool

	VerifyPKCE(code *SkygearAuthorizationCode, codeVerifier string) error
}
