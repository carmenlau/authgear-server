package webapp

import (
	"github.com/google/wire"
)

var DependencySet = wire.NewSet(
	wire.Struct(new(URLProvider), "*"),
	wire.Struct(new(AuthenticateURLProvider), "*"),

	NewCSRFCookieDef,
	NewSessionCookieDef,
	NewErrorCookieDef,
	wire.Struct(new(ErrorCookie), "*"),

	wire.Struct(new(CSRFMiddleware), "*"),
	wire.Struct(new(AuthEntryPointMiddleware), "*"),
	wire.Struct(new(SessionMiddleware), "*"),
	wire.Bind(new(SessionMiddlewareStore), new(*SessionStoreRedis)),

	NewServiceLogger,
	wire.Struct(new(SessionStoreRedis), "*"),
	wire.Bind(new(SessionStore), new(*SessionStoreRedis)),
	wire.Struct(new(Service2), "*"),
	wire.Bind(new(PageService), new(*Service2)),
)
