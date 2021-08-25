package instrument

import (
	"context"
	"net/http"

	"github.com/authgear/authgear-server/pkg/lib/config"
	"github.com/authgear/authgear-server/pkg/lib/session"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
)

type PageViewMiddleware struct {
	AppID config.AppID
}

func (m *PageViewMiddleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.Background()
		sess := session.GetSession(r.Context())
		userID := ""
		if sess != nil {
			userID = sess.GetAuthenticationInfo().UserID
		}
		labels := []attribute.KeyValue{
			attribute.String("userID", userID),
			attribute.String("appID", string(m.AppID)),
			attribute.String("path", r.URL.Path),
		}
		meter := global.Meter("authgear-main-meter")
		counter := metric.Must(meter).
			NewInt64Counter("view_count").
			Bind(labels...)
		counter.Add(ctx, 1)
		next.ServeHTTP(w, r)
	})
}
