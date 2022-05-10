package webapp

import (
	"github.com/authgear/authgear-server/pkg/auth/webapp"
	"github.com/authgear/authgear-server/pkg/util/clock"
)

type SessionStore interface {
	Get(id string) (session *webapp.Session, err error)
	Update(session *webapp.Session) (err error)
}

type SessionProvider struct {
	SessionStore SessionStore
	Publisher    *Publisher
	Clock        clock.Clock
}

func (p *SessionProvider) GetSession(sessionID string) (session *webapp.Session, err error) {
	return p.SessionStore.Get(sessionID)
}

func (p *SessionProvider) UpdateSession(s *webapp.Session) error {
	now := p.Clock.NowUTC()
	s.UpdatedAt = now
	err := p.SessionStore.Update(s)
	if err != nil {
		return err
	}

	msg := &WebsocketMessage{
		Kind: WebsocketMessageKindRefresh,
	}

	err = p.Publisher.Publish(s, msg)
	if err != nil {
		return err
	}

	return nil
}
