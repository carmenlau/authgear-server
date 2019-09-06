package mfa

import (
	"net/http"

	"github.com/skygeario/skygear-server/pkg/auth"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/mfa"
	coreAuth "github.com/skygeario/skygear-server/pkg/core/auth"
	"github.com/skygeario/skygear-server/pkg/core/auth/authz"
	"github.com/skygeario/skygear-server/pkg/core/auth/authz/policy"
	"github.com/skygeario/skygear-server/pkg/core/db"
	"github.com/skygeario/skygear-server/pkg/core/handler"
	"github.com/skygeario/skygear-server/pkg/core/inject"
	"github.com/skygeario/skygear-server/pkg/core/server"
	"github.com/skygeario/skygear-server/pkg/core/skyerr"
)

func AttachDeleteAuthenticatorHandler(
	server *server.Server,
	authDependency auth.DependencyMap,
) *server.Server {
	server.Handle("/mfa/authenticator/delete", &DeleteAuthenticatorHandlerFactory{
		Dependency: authDependency,
	}).Methods("OPTIONS", "POST")
	return server
}

type DeleteAuthenticatorHandlerFactory struct {
	Dependency auth.DependencyMap
}

func (f DeleteAuthenticatorHandlerFactory) NewHandler(request *http.Request) http.Handler {
	h := &DeleteAuthenticatorHandler{}
	inject.DefaultRequestInject(h, f.Dependency, request)
	return handler.RequireAuthz(handler.APIHandlerToHandler(h, h.TxContext), h.AuthContext, h)
}

type DeleteAuthenticatorRequest struct {
	AuthenticatorID string `json:"authenticator_id"`
}

func (r DeleteAuthenticatorRequest) Validate() error {
	if r.AuthenticatorID == "" {
		return skyerr.NewInvalidArgument("missing authenticator ID", []string{"authenticator_id"})
	}
	return nil
}

/*
	@Operation POST /mfa/authenticator/delete - Delete authenticator.
		Delete authenticator.

		@Tag User
		@SecurityRequirement access_key
		@SecurityRequirement access_token

		@RequestBody {DeleteAuthenticatorRequest}
		@Response 200 {EmptyResponse}
*/
type DeleteAuthenticatorHandler struct {
	TxContext   db.TxContext           `dependency:"TxContext"`
	AuthContext coreAuth.ContextGetter `dependency:"AuthContextGetter"`
	MFAProvider mfa.Provider           `dependency:"MFAProvider"`
}

func (h *DeleteAuthenticatorHandler) ProvideAuthzPolicy() authz.Policy {
	return policy.AllOf(
		authz.PolicyFunc(policy.DenyNoAccessKey),
		authz.PolicyFunc(policy.RequireAuthenticated),
		authz.PolicyFunc(policy.DenyDisabledUser),
	)
}

func (h *DeleteAuthenticatorHandler) WithTx() bool {
	return true
}

func (h *DeleteAuthenticatorHandler) DecodeRequest(request *http.Request) (handler.RequestPayload, error) {
	payload := DeleteAuthenticatorRequest{}
	err := handler.DecodeJSONBody(request, &payload)
	return payload, err
}

func (h *DeleteAuthenticatorHandler) Handle(req interface{}) (resp interface{}, err error) {
	payload := req.(DeleteAuthenticatorRequest)
	userID := h.AuthContext.AuthInfo().ID
	err = h.MFAProvider.DeleteAuthenticator(userID, payload.AuthenticatorID)
	if err != nil {
		return
	}
	resp = map[string]interface{}{}
	return
}
