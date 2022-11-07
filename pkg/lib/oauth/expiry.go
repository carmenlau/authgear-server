package oauth

import (
	"errors"
	"time"

	"github.com/authgear/authgear-server/pkg/lib/config"
	"github.com/authgear/authgear-server/pkg/lib/session/idpsession"
)

type ExpiryServiceIDPSessionProvider interface {
	Get(id string) (*idpsession.IDPSession, error)
	ComputeSessionExpiry(session *idpsession.IDPSession) time.Time
}

type OfflineGrantExpiryService struct {
	OAuthConfig *config.OAuthConfig
	IDPSessions ExpiryServiceIDPSessionProvider
}

func (s *OfflineGrantExpiryService) ComputeOfflineGrantExpiryWithClients(session *OfflineGrant) (expiry time.Time, err error) {
	var clientConfig *config.OAuthClientConfig
	for _, c := range s.OAuthConfig.Clients {
		if c.ClientID == session.ClientID {
			cc := c
			clientConfig = &cc
		}
	}

	if clientConfig == nil {
		err = ErrGrantNotFound
		return
	}

	expiry, err = s.ComputeOfflineGrantExpiryWithClient(session, clientConfig)
	return
}

func (s *OfflineGrantExpiryService) ComputeOfflineGrantExpiryWithClient(session *OfflineGrant, cfg *config.OAuthClientConfig) (expiry time.Time, err error) {
	expiry = session.CreatedAt.Add(cfg.RefreshTokenLifetime.Duration())
	if *cfg.RefreshTokenIdleTimeoutEnabled {
		idleExpiry := session.AccessInfo.LastAccess.Timestamp.Add(cfg.RefreshTokenIdleTimeout.Duration())
		if idleExpiry.Before(expiry) {
			expiry = idleExpiry
		}
	}

	if session.SSOEnabled {
		if session.IDPSessionID == "" {
			// adjust the expiry with best effort
			// should not block the flow if idp session doesn't exist
			return
		}

		idp, e := s.IDPSessions.Get(session.IDPSessionID)
		if err != nil {
			if errors.Is(e, idpsession.ErrSessionNotFound) {
				// adjust the expiry with best effort
				// should not block the flow if idp session doesn't exist
				return
			}
			err = e
			return
		}

		idpSessionExpiry := s.IDPSessions.ComputeSessionExpiry(idp)
		if idpSessionExpiry.Before(expiry) {
			expiry = idpSessionExpiry
		}
	}
	return
}
