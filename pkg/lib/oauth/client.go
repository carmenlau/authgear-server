package oauth

import (
	"github.com/authgear/authgear-server/pkg/lib/session"
	"github.com/authgear/authgear-server/pkg/lib/session/idpsession"
)

func ClientID(s session.Session) string {
	switch s := s.(type) {
	case *idpsession.IDPSession:
		return ""
	case *OfflineGrant:
		return s.ClientID
	default:
		panic("oauth: unexpected session type for client id")
	}
}
