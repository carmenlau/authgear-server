//go:build wireinject
// +build wireinject

package webapp

import (
	"github.com/authgear/authgear-server/pkg/auth/webapp"
	"github.com/authgear/authgear-server/pkg/lib/config"
	"github.com/authgear/authgear-server/pkg/lib/infra/redis/appredis"
	"github.com/authgear/authgear-server/pkg/util/clock"
	"github.com/google/wire"
)

func newSessionProvider(appID config.AppID, clock clock.Clock, redisHandle *appredis.Handle) *SessionProvider {
	panic(wire.Build(
		webapp.DependencySet,
		NewPublisher,
		wire.Struct(new(SessionProvider), "*"),
		wire.Bind(new(SessionStore), new(*webapp.SessionStoreRedis)),
	))
}
