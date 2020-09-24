package transport

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/graphql-go/relay"

	"github.com/authgear/authgear-server/pkg/lib/config"
	"github.com/authgear/authgear-server/pkg/util/httproute"
	"github.com/authgear/authgear-server/pkg/util/log"
)

func ConfigureAdminAPIRoute(route httproute.Route) httproute.Route {
	return route.WithMethods("OPTIONS", "GET", "POST").WithPathPattern("/api/apps/:appid/graphql")
}

type AdminAPIConfigResolver interface {
	ResolveConfig(appID string) (*config.Config, error)
}

type AdminAPIEndpointResolver interface {
	ResolveEndpoint(appID string) (url *url.URL, err error)
}

type AdminAPIAuthzAdder interface {
	AddAuthz(appID config.AppID, authKey *config.AdminAPIAuthKey, hdr http.Header) (err error)
}

type AdminAPILogger struct{ *log.Logger }

func NewAdminAPILogger(lf *log.Factory) AdminAPILogger {
	return AdminAPILogger{lf.New("admin-api-proxy")}
}

type AdminAPIHandler struct {
	ConfigResolver   AdminAPIConfigResolver
	EndpointResolver AdminAPIEndpointResolver
	AuthzAdder       AdminAPIAuthzAdder
	Logger           AdminAPILogger
}

func (h *AdminAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	resolved := relay.FromGlobalID(httproute.GetParam(r, "appid"))
	if resolved == nil || resolved.Type != "App" {
		h.Logger.Debugf("invalid app ID: %v", resolved)
		http.Error(w, "invalid app ID", http.StatusBadRequest)
		return
	}

	appID := resolved.ID

	cfg, err := h.ConfigResolver.ResolveConfig(appID)
	if err != nil {
		h.Logger.WithError(err).Debugf("failed to resolve config")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	hostURL, err := url.Parse(cfg.AppConfig.HTTP.PublicOrigin)
	if err != nil {
		h.Logger.WithError(err).Debugf("failed to parse public origin: %v", cfg.AppConfig.HTTP.PublicOrigin)
		http.Error(w, fmt.Errorf("failed to parse public origin: %v", cfg.AppConfig.HTTP.PublicOrigin).Error(), http.StatusInternalServerError)
		return
	}

	authKey, ok := cfg.SecretConfig.LookupData(config.AdminAPIAuthKeyKey).(*config.AdminAPIAuthKey)
	if !ok {
		h.Logger.WithError(err).Debugf("failed to look up admin API auth key")
		http.Error(w, "failed to look up admin API auth key", http.StatusInternalServerError)
		return
	}

	endpoint, err := h.EndpointResolver.ResolveEndpoint(appID)
	if err != nil {
		h.Logger.WithError(err).Debugf("failed to resolve endpoint")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.Logger.WithFields(map[string]interface{}{
		"appID":    appID,
		"hostURL":  hostURL.String(),
		"endpoint": endpoint.String(),
		"host":     hostURL.Host,
	}).Debugf("proxy admin API")

	proxy := httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL = endpoint
			// We have to set both to ensure Admin API server see the correct host.
			req.Host = hostURL.Host
			req.Header.Set("X-Forwarded-Host", req.Host)
			err = h.AuthzAdder.AddAuthz(
				config.AppID(appID),
				authKey,
				req.Header,
			)
			if err != nil {
				panic(err)
			}
		},
	}

	proxy.ServeHTTP(w, r)
}
