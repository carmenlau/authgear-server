package clientid

import (
	"net/http"
)

func GetClientID(r *http.Request) (clientID string) {
	if r == nil {
		return ""
	}
	if r.URL == nil {
		return ""
	}
	return r.URL.Query().Get("client_id")
}
