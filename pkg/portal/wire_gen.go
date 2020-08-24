// Code generated by Wire. DO NOT EDIT.

//go:generate wire
//+build !wireinject

package portal

import (
	"github.com/authgear/authgear-server/pkg/lib/infra/middleware"
	"github.com/authgear/authgear-server/pkg/portal/deps"
	"github.com/authgear/authgear-server/pkg/portal/graphql"
	"github.com/authgear/authgear-server/pkg/portal/loader"
	"github.com/authgear/authgear-server/pkg/portal/service"
	"github.com/authgear/authgear-server/pkg/portal/session"
	"github.com/authgear/authgear-server/pkg/portal/transport"
	"github.com/authgear/authgear-server/pkg/util/httproute"
	"net/http"
)

// Injectors from wire.go:

func newRecoverMiddleware(p *deps.RequestProvider) httproute.Middleware {
	rootProvider := p.RootProvider
	factory := rootProvider.LoggerFactory
	recoveryLogger := middleware.NewRecoveryLogger(factory)
	recoverMiddleware := &middleware.RecoverMiddleware{
		Logger: recoveryLogger,
	}
	return recoverMiddleware
}

func newSessionInfoMiddleware(p *deps.RequestProvider) httproute.Middleware {
	sessionMiddleware := &session.Middleware{}
	return sessionMiddleware
}

func newGraphQLHandler(p *deps.RequestProvider) (http.Handler, error) {
	rootProvider := p.RootProvider
	serverConfig := rootProvider.ServerConfig
	request := p.Request
	context := deps.ProvideRequestContext(request)
	viewerLoader := &loader.ViewerLoader{
		Context: context,
	}
	config, err := service.NewLibConfig(serverConfig)
	if err != nil {
		return nil, err
	}
	appService := &service.AppService{
		Config: config,
	}
	appLoader := &loader.AppLoader{
		Apps: appService,
	}
	graphqlContext := &graphql.Context{
		Viewer: viewerLoader,
		Apps:   appLoader,
	}
	graphQLHandler := &transport.GraphQLHandler{
		Config:         serverConfig,
		GraphQLContext: graphqlContext,
	}
	return graphQLHandler, nil
}

func newRuntimeConfigHandler(p *deps.RequestProvider) http.Handler {
	rootProvider := p.RootProvider
	serverConfig := rootProvider.ServerConfig
	runtimeConfigHandler := &transport.RuntimeConfigHandler{
		Config: serverConfig,
	}
	return runtimeConfigHandler
}