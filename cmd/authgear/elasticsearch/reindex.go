package elasticsearch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"

	"github.com/authgear/authgear-server/pkg/api/model"
	identityloginid "github.com/authgear/authgear-server/pkg/lib/authn/identity/loginid"
	identityoauth "github.com/authgear/authgear-server/pkg/lib/authn/identity/oauth"
	"github.com/authgear/authgear-server/pkg/lib/authn/user"
	"github.com/authgear/authgear-server/pkg/lib/config"
	libes "github.com/authgear/authgear-server/pkg/lib/elasticsearch"
	"github.com/authgear/authgear-server/pkg/lib/infra/db"
)

type Reindexer struct {
	AppID   config.AppID
	Users   *user.Store
	OAuth   *identityoauth.Store
	LoginID *identityloginid.Store
}

func (q *Reindexer) QueryPage(after model.PageCursor, first uint64) ([]model.PageItem, error) {
	users, offset, err := q.Users.QueryPage(after, model.PageCursor(""), &first, nil)
	if err != nil {
		return nil, err
	}

	models := make([]model.PageItem, len(users))
	for i, u := range users {
		pageKey := db.PageKey{Offset: offset + uint64(i)}
		cursor, err := pageKey.ToPageCursor()
		if err != nil {
			return nil, err
		}
		oauthIdentities, err := q.OAuth.List(u.ID)
		if err != nil {
			return nil, err
		}
		loginIDIdentities, err := q.LoginID.List(u.ID)
		if err != nil {
			return nil, err
		}
		val := &libes.User{
			ID:          u.ID,
			AppID:       string(q.AppID),
			CreatedAt:   u.CreatedAt,
			UpdatedAt:   u.UpdatedAt,
			LastLoginAt: u.LastLoginAt,
			IsDisabled:  u.IsDisabled,
		}

		var arrClaims []map[string]interface{}
		for _, oauthI := range oauthIdentities {
			arrClaims = append(arrClaims, oauthI.Claims)
		}
		for _, loginIDI := range loginIDIdentities {
			arrClaims = append(arrClaims, loginIDI.Claims)
		}

		for _, claims := range arrClaims {
			email, ok := claims["email"].(string)
			if ok {
				val.Email = append(val.Email, email)
			}
			phoneNumber, ok := claims["phone_number"].(string)
			if ok {
				val.PhoneNumber = append(val.PhoneNumber, phoneNumber)
			}
			preferredUsername, ok := claims["preferred_username"].(string)
			if ok {
				val.PreferredUsername = append(val.PreferredUsername, preferredUsername)
			}
		}

		models[i] = model.PageItem{Value: val, Cursor: cursor}
	}

	return models, nil
}

func (q *Reindexer) Reindex(es *elasticsearch.Client) (err error) {
	var first uint64 = 50
	var after model.PageCursor = ""
	var items []model.PageItem

	for {
		items, err = q.QueryPage(after, first)
		if err != nil {
			return
		}

		// Termination condition
		if len(items) <= 0 {
			break
		}

		// Prepare for next iteration
		after = items[len(items)-1].Cursor

		// Process the items
		buf := &bytes.Buffer{}
		for _, item := range items {
			user := item.Value.(*libes.User)
			fmt.Printf("Indexing app (%s) user (%s)\n", user.AppID, user.ID)
			err = q.writeBody(buf, user)
			if err != nil {
				return
			}
		}

		var res *esapi.Response
		res, err = es.Bulk(buf, func(o *esapi.BulkRequest) {
			o.Index = libes.IndexNameUser
		})
		if err != nil {
			return
		}
		defer res.Body.Close()
		if res.IsError() {
			err = fmt.Errorf("%v", res)
			return
		}
	}

	return nil
}

func (q *Reindexer) writeBody(buf io.Writer, user *libes.User) (err error) {
	id := fmt.Sprintf("%s:%s", user.AppID, user.ID)
	action := map[string]interface{}{
		"index": map[string]interface{}{
			"_id": id,
		},
	}
	actionBytes, err := json.Marshal(action)
	if err != nil {
		return
	}

	_, err = buf.Write(actionBytes)
	if err != nil {
		return
	}

	_, err = buf.Write([]byte("\n"))
	if err != nil {
		return
	}

	sourceBytes, err := json.Marshal(user)
	if err != nil {
		return
	}

	_, err = buf.Write(sourceBytes)
	if err != nil {
		return
	}

	_, err = buf.Write([]byte("\n"))
	if err != nil {
		return
	}

	return nil
}
