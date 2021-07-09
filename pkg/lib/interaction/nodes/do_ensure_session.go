package nodes

import (
	"net/http"

	"github.com/authgear/authgear-server/pkg/api/event/nonblocking"
	"github.com/authgear/authgear-server/pkg/api/model"
	"github.com/authgear/authgear-server/pkg/lib/interaction"
	"github.com/authgear/authgear-server/pkg/lib/session"
	"github.com/authgear/authgear-server/pkg/lib/session/idpsession"
)

func init() {
	interaction.RegisterNode(&NodeDoEnsureSession{})
}

type EnsureSessionMode string

const (
	EnsureSessionModeDefault          EnsureSessionMode = ""
	EnsureSessionModeCreate           EnsureSessionMode = "create"
	EnsureSessionModeUpdateIfPossible EnsureSessionMode = "update_if_possible"
	EnsureSessionModeNoop             EnsureSessionMode = "noop"
)

type EdgeDoEnsureSession struct {
	CreateReason session.CreateReason
	Mode         EnsureSessionMode
}

func (e *EdgeDoEnsureSession) Instantiate(ctx *interaction.Context, graph *interaction.Graph, input interface{}) (interaction.Node, error) {
	amr := graph.GetAMR()
	userID := graph.MustGetUserID()

	mode := e.Mode
	if mode == EnsureSessionModeDefault {
		mode = EnsureSessionModeCreate
	}

	attrs := session.NewAttrs(userID)
	attrs.SetAMR(amr)
	sessionToCreate, token := ctx.Sessions.MakeSession(attrs)
	sessionCookie := ctx.CookieFactory.ValueCookie(ctx.SessionCookie.Def, token)

	var sessionToUpdate *idpsession.IDPSession
	if mode == EnsureSessionModeUpdateIfPossible {
		s := session.GetSession(ctx.Request.Context())
		if idp, ok := s.(*idpsession.IDPSession); ok && idp.GetUserID() == userID {
			ctx.Sessions.Reauthenticate(idp, amr)

			sessionToUpdate = idp
			sessionToCreate = nil
			sessionCookie = nil
		}
	}

	if mode == EnsureSessionModeNoop {
		sessionToCreate = nil
		sessionToUpdate = nil
		sessionCookie = nil
	}

	sameSiteStrictCookie := ctx.CookieFactory.ValueCookie(
		ctx.SessionCookie.SameSiteStrictDef,
		"true",
	)

	return &NodeDoEnsureSession{
		CreateReason:         e.CreateReason,
		SessionToCreate:      sessionToCreate,
		SessionToUpdate:      sessionToUpdate,
		SessionCookie:        sessionCookie,
		SameSiteStrictCookie: sameSiteStrictCookie,
		IsAdminAPI:           interaction.IsAdminAPI(input),
	}, nil
}

type NodeDoEnsureSession struct {
	CreateReason         session.CreateReason   `json:"reason"`
	SessionToCreate      *idpsession.IDPSession `json:"session_to_create,omitempty"`
	SessionToUpdate      *idpsession.IDPSession `json:"session_to_update,omitempty"`
	SessionCookie        *http.Cookie           `json:"session_cookie,omitempty"`
	SameSiteStrictCookie *http.Cookie           `json:"same_site_strict_cookie,omitempty"`
	IsAdminAPI           bool                   `json:"is_admin_api"`
}

// GetCookies implements CookiesGetter
func (n *NodeDoEnsureSession) GetCookies() (cookies []*http.Cookie) {
	if n.SessionCookie != nil {
		cookies = append(cookies, n.SessionCookie)
	}
	if n.SameSiteStrictCookie != nil {
		cookies = append(cookies, n.SameSiteStrictCookie)
	}
	return
}

func (n *NodeDoEnsureSession) Prepare(ctx *interaction.Context, graph *interaction.Graph) error {
	return nil
}

func (n *NodeDoEnsureSession) GetSession() (*idpsession.IDPSession, bool) {
	if n.SessionToCreate != nil {
		return n.SessionToCreate, true
	}
	if n.SessionToUpdate != nil {
		return n.SessionToUpdate, true
	}
	return nil, false
}

func (n *NodeDoEnsureSession) GetEffects() ([]interaction.Effect, error) {
	return []interaction.Effect{
		interaction.EffectOnCommit(func(ctx *interaction.Context, graph *interaction.Graph, nodeIndex int) error {
			if n.CreateReason != session.CreateReasonPromote {
				return nil
			}

			userID := graph.MustGetUserID()

			newUser, err := ctx.Users.Get(userID)
			if err != nil {
				return err
			}

			anonUser := newUser
			if identityCheck, ok := getIdentityConflictNode(graph); ok && identityCheck.DuplicatedIdentity != nil {
				// Logging as existing user when promoting: old user is different.
				anonUser, err = ctx.Users.Get(identityCheck.NewIdentity.UserID)
				if err != nil {
					return err
				}
			}

			var identities []model.Identity
			for _, info := range graph.GetUserNewIdentities() {
				identities = append(identities, info.ToModel())
			}

			err = ctx.Events.DispatchEvent(&nonblocking.UserAnonymousPromotedEventPayload{
				AnonymousUser: *anonUser,
				User:          *newUser,
				Identities:    identities,
				AdminAPI:      n.IsAdminAPI,
			})
			if err != nil {
				return err
			}

			return nil
		}),
		interaction.EffectOnCommit(func(ctx *interaction.Context, graph *interaction.Graph, nodeIndex int) error {
			userID := graph.MustGetUserID()

			var err error
			if sess, ok := n.GetSession(); ok {
				err = ctx.Users.UpdateLoginTime(userID, sess.AuthenticatedAt)
				if err != nil {
					return err
				}
			}

			user, err := ctx.Users.Get(userID)
			if err != nil {
				return err
			}

			if n.SessionToCreate != nil {
				if n.CreateReason == session.CreateReasonLogin || n.CreateReason == session.CreateReasonReauthenticate {
					err = ctx.Events.DispatchEvent(&nonblocking.UserAuthenticatedEventPayload{
						User:     *user,
						Session:  *n.SessionToCreate.ToAPIModel(),
						AdminAPI: n.IsAdminAPI,
					})
					if err != nil {
						return err
					}
				}
			}

			if n.SessionToCreate != nil {
				err = ctx.Sessions.Create(n.SessionToCreate)
				if err != nil {
					return err
				}

				// Clean up unreachable IdP Session.
				s := session.GetSession(ctx.Request.Context())
				if s != nil && s.SessionType() == session.TypeIdentityProvider {
					err = ctx.SessionManager.Delete(s)
					if err != nil {
						return err
					}
				}
			}

			if n.SessionToUpdate != nil {
				err = ctx.Sessions.Update(n.SessionToUpdate)
				if err != nil {
					return err
				}
			}

			return nil
		}),
		interaction.EffectOnCommit(func(ctx *interaction.Context, graph *interaction.Graph, nodeIndex int) error {
			userID := graph.MustGetUserID()
			err := ctx.Search.ReindexUser(userID, false)
			if err != nil {
				return err
			}
			return nil
		}),
	}, nil
}

func (n *NodeDoEnsureSession) DeriveEdges(graph *interaction.Graph) ([]interaction.Edge, error) {
	return graph.Intent.DeriveEdgesForNode(graph, n)
}