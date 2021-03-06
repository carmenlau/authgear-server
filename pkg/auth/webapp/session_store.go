package webapp

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	goredis "github.com/gomodule/redigo/redis"

	"github.com/authgear/authgear-server/pkg/lib/config"
	"github.com/authgear/authgear-server/pkg/lib/infra/redis"
)

const SessionExpiryDuration = 5 * time.Minute

type SessionStoreRedis struct {
	AppID config.AppID
	Redis *redis.Handle
}

func (s *SessionStoreRedis) Create(session *Session) (err error) {
	key := sessionKey(string(s.AppID), session.ID)
	bytes, err := json.Marshal(session)
	if err != nil {
		return
	}

	err = s.Redis.WithConn(func(conn redis.Conn) error {
		ttl := toMilliseconds(SessionExpiryDuration)
		_, err = goredis.String(conn.Do("SET", key, bytes, "PX", ttl, "NX"))
		if errors.Is(err, goredis.ErrNil) {
			return fmt.Errorf("webapp-store: failed to create session: %w", err)
		}
		if err != nil {
			return err
		}
		return nil
	})
	return
}

func (s *SessionStoreRedis) Update(session *Session) (err error) {
	key := sessionKey(string(s.AppID), session.ID)
	bytes, err := json.Marshal(session)
	if err != nil {
		return
	}

	err = s.Redis.WithConn(func(conn redis.Conn) error {
		ttl := toMilliseconds(SessionExpiryDuration)
		_, err = goredis.String(conn.Do("SET", key, bytes, "PX", ttl, "XX"))
		if errors.Is(err, goredis.ErrNil) {
			return ErrInvalidSession
		}
		if err != nil {
			return err
		}
		return nil
	})
	return
}

func (s *SessionStoreRedis) Get(id string) (session *Session, err error) {
	key := sessionKey(string(s.AppID), id)
	err = s.Redis.WithConn(func(conn redis.Conn) error {
		data, err := goredis.Bytes(conn.Do("GET", key))
		if errors.Is(err, goredis.ErrNil) {
			err = ErrInvalidSession
			return err
		}
		if err != nil {
			return err
		}
		err = json.Unmarshal(data, &session)
		if err != nil {
			return err
		}

		return nil
	})
	return
}

func (s *SessionStoreRedis) Delete(id string) error {
	key := sessionKey(string(s.AppID), id)
	err := s.Redis.WithConn(func(conn redis.Conn) error {
		_, err := conn.Do("DEL", key)
		return err
	})
	return err
}

func toMilliseconds(d time.Duration) int64 {
	return int64(d / time.Millisecond)
}

func sessionKey(appID string, id string) string {
	return fmt.Sprintf("app:%s:webapp-session:%s", appID, id)
}
