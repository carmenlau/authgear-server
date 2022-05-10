package webapp

import (
	"fmt"
	"net/http"

	"github.com/authgear/authgear-server/pkg/lib/config"
	"github.com/authgear/authgear-server/pkg/lib/infra/redis/appredis"
	"github.com/authgear/authgear-server/pkg/util/clock"
	"github.com/authgear/authgear-server/pkg/util/httproute"
	"github.com/authgear/authgear-server/pkg/util/log"
)

func ConfigureWhatsappWATICallbackRoute(route httproute.Route) httproute.Route {
	return route.
		WithMethods("OPTIONS", "GET").
		WithPathPattern("/whatsapp/callback/wati")
}

type WhatsappWATICallbackHandler struct {
	ControllerFactory    ControllerFactory
	WhatsappCodeProvider WhatsappCodeProvider
	Logger               WhatsappWATICallbackHandlerLogger

	// dependencies of newSessionProvider
	Clock       clock.Clock
	RedisHandle *appredis.Handle
}

type WhatsappWATICallbackHandlerLogger struct{ *log.Logger }

func NewWhatsappWATICallbackHandlerLogger(lf *log.Factory) WhatsappWATICallbackHandlerLogger {
	return WhatsappWATICallbackHandlerLogger{lf.New("webapp-whatsapp-wati-callback-handler")}
}

func (h *WhatsappWATICallbackHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctrl, err := h.ControllerFactory.New(r, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer ctrl.Serve()

	// fixme(whatsapp): implement the real webhook
	handler := func() error {
		var err error
		defer func() {
			if err != nil {
				h.Logger.WithError(err).Error("failed to consume message")
			}
		}()
		// http://accounts.portal.localhost:3100/whatsapp/callback/wati?phone=%2B93701234567&code=123456
		phone := r.Form.Get("phone")
		code := r.Form.Get("code")
		fmt.Println("phone", phone, code)
		codeModel, err := h.WhatsappCodeProvider.SetUserInputtedCode(phone, code)
		if err != nil {
			return nil
		}

		// Update the web session and trigger the refresh event
		webSessionProvider := newSessionProvider(config.AppID(codeModel.AppID), h.Clock, h.RedisHandle)
		webSession, err := webSessionProvider.GetSession(codeModel.WebSessionID)
		if err != nil {
			return nil
		}
		err = webSessionProvider.UpdateSession(webSession)
		if err != nil {
			return nil
		}
		return nil
	}
	ctrl.Get(handler)
}
