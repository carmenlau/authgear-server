package newinteraction

import (
	"net/http"
	"time"

	"github.com/authgear/authgear-server/pkg/auth/config"
	"github.com/authgear/authgear-server/pkg/auth/dependency/authenticator"
	"github.com/authgear/authgear-server/pkg/auth/dependency/challenge"
	"github.com/authgear/authgear-server/pkg/auth/dependency/identity"
	"github.com/authgear/authgear-server/pkg/auth/dependency/identity/anonymous"
	"github.com/authgear/authgear-server/pkg/auth/dependency/identity/loginid"
	"github.com/authgear/authgear-server/pkg/auth/dependency/session"
	"github.com/authgear/authgear-server/pkg/auth/dependency/sso"
	"github.com/authgear/authgear-server/pkg/auth/dependency/user"
	"github.com/authgear/authgear-server/pkg/auth/dependency/verification"
	"github.com/authgear/authgear-server/pkg/auth/event"
	"github.com/authgear/authgear-server/pkg/auth/model"
	"github.com/authgear/authgear-server/pkg/clock"
	"github.com/authgear/authgear-server/pkg/core/authn"
	"github.com/authgear/authgear-server/pkg/db"
	"github.com/authgear/authgear-server/pkg/httputil"
	"github.com/authgear/authgear-server/pkg/mfa"
	"github.com/authgear/authgear-server/pkg/otp"
)

type IdentityService interface {
	Get(userID string, typ authn.IdentityType, id string) (*identity.Info, error)
	GetBySpec(spec *identity.Spec) (*identity.Info, error)
	ListByUser(userID string) ([]*identity.Info, error)
	New(userID string, spec *identity.Spec) (*identity.Info, error)
	UpdateWithSpec(is *identity.Info, spec *identity.Spec) (*identity.Info, error)
	Create(is *identity.Info) error
	Update(is *identity.Info) error
	Delete(is *identity.Info) error
	CheckDuplicated(info *identity.Info) (*identity.Info, error)
}

type AuthenticatorService interface {
	Get(userID string, typ authn.AuthenticatorType, id string) (*authenticator.Info, error)
	List(userID string, filters ...authenticator.Filter) ([]*authenticator.Info, error)
	New(spec *authenticator.Spec, secret string) (*authenticator.Info, error)
	WithSecret(authenticatorInfo *authenticator.Info, secret string) (changed bool, info *authenticator.Info, err error)
	Create(authenticatorInfo *authenticator.Info) error
	Update(authenticatorInfo *authenticator.Info) error
	Delete(authenticatorInfo *authenticator.Info) error
	VerifySecret(info *authenticator.Info, state map[string]string, secret string) error
}

type OOBAuthenticatorProvider interface {
	GenerateCode(secret string, channel authn.AuthenticatorOOBChannel) string
	SendCode(
		channel authn.AuthenticatorOOBChannel,
		target string,
		code string,
		messageType otp.MessageType,
	) (*otp.CodeSendResult, error)
}

type AnonymousIdentityProvider interface {
	ParseRequestUnverified(requestJWT string) (*anonymous.Request, error)
	GetByKeyID(keyID string) (*anonymous.Identity, error)
	ParseRequest(requestJWT string, identity *anonymous.Identity) (*anonymous.Request, error)
}

type ChallengeProvider interface {
	Consume(token string) (*challenge.Purpose, error)
}

type MFAService interface {
	GenerateDeviceToken() string
	CreateDeviceToken(userID string, token string) (*mfa.DeviceToken, error)
	VerifyDeviceToken(userID string, token string) error
	InvalidateAllDeviceTokens(userID string) error

	GenerateRecoveryCodes() []string
	ReplaceRecoveryCodes(userID string, codes []string) ([]*mfa.RecoveryCode, error)
	ConsumeRecoveryCode(userID string, code string) error
	ListRecoveryCodes(userID string) ([]*mfa.RecoveryCode, error)
}

type UserService interface {
	Get(id string) (*model.User, error)
	Create(userID string, metadata map[string]interface{}) (*user.User, error)
	AfterCreate(user *user.User, identities []*identity.Info, authenticators []*authenticator.Info) error
	UpdateLoginTime(user *model.User, lastLoginAt time.Time) error
}

type HookProvider interface {
	DispatchEvent(payload event.Payload, user *model.User) error
}

type SessionProvider interface {
	MakeSession(*authn.Attrs) (*session.IDPSession, string)
	Create(*session.IDPSession) error
}

type OAuthProviderFactory interface {
	NewOAuthProvider(alias string) sso.OAuthProvider
}

type ForgotPasswordService interface {
	SendCode(loginID string) error
}

type ResetPasswordService interface {
	ResetPassword(code string, newPassword string) (oldInfo *authenticator.Info, newInfo *authenticator.Info, err error)
	HashCode(code string) (codeHash string)
	AfterResetPassword(codeHash string) error
}

type LoginIDNormalizerFactory interface {
	NormalizerWithLoginIDType(loginIDKeyType config.LoginIDKeyType) loginid.Normalizer
}

type VerificationService interface {
	GetVerificationStatus(i *identity.Info) (verification.Status, error)
	CreateNewCode(id string, info *identity.Info) (*verification.Code, error)
	GetCode(id string) (*verification.Code, error)
	VerifyCode(id string, code string) (*verification.Code, error)
	NewVerificationAuthenticator(code *verification.Code) (*authenticator.Info, error)
	SendCode(code *verification.Code, webStateID string) (*otp.CodeSendResult, error)
}

type CookieFactory interface {
	ValueCookie(def *httputil.CookieDef, value string) *http.Cookie
}

type Context struct {
	IsDryRun   bool   `wire:"-"`
	WebStateID string `wire:"-"`

	Database db.SQLExecutor
	Clock    clock.Clock
	Config   *config.AppConfig

	Identities               IdentityService
	Authenticators           AuthenticatorService
	AnonymousIdentities      AnonymousIdentityProvider
	OOBAuthenticators        OOBAuthenticatorProvider
	OAuthProviderFactory     OAuthProviderFactory
	MFA                      MFAService
	ForgotPassword           ForgotPasswordService
	ResetPassword            ResetPasswordService
	LoginIDNormalizerFactory LoginIDNormalizerFactory
	Verification             VerificationService

	Challenges    ChallengeProvider
	Users         UserService
	Hooks         HookProvider
	CookieFactory CookieFactory
	Sessions      SessionProvider
	SessionCookie session.CookieDef
}

var interactionGraphSavePoint savePoint = "interaction_graph"

func (c *Context) initialize() (*Context, error) {
	ctx := *c
	_, err := ctx.Database.ExecWith(interactionGraphSavePoint.New())
	return &ctx, err
}

func (c *Context) commit() error {
	_, err := c.Database.ExecWith(interactionGraphSavePoint.Release())
	return err
}

func (c *Context) rollback() error {
	_, err := c.Database.ExecWith(interactionGraphSavePoint.Rollback())
	return err
}

func (c *Context) perform(effect Effect) error {
	return effect.apply(c)
}