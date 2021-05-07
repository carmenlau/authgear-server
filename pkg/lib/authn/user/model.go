package user

import (
	"fmt"
	"time"

	"github.com/authgear/authgear-server/pkg/api/model"
	"github.com/authgear/authgear-server/pkg/lib/authn"
	"github.com/authgear/authgear-server/pkg/lib/authn/identity"
	"github.com/authgear/authgear-server/pkg/lib/infra/db"
)

type SortBy string

const (
	SortByDefault     SortBy = ""
	SortByCreatedAt   SortBy = "created_at"
	SortByLastLoginAt SortBy = "last_login_at"
)

type SortOption struct {
	SortBy        SortBy
	SortDirection model.SortDirection
}

func (o SortOption) Apply(builder db.SelectBuilder) db.SelectBuilder {
	sortBy := o.SortBy
	if sortBy == SortByDefault {
		sortBy = SortByCreatedAt
	}

	sortDirection := o.SortDirection
	if sortDirection == model.SortDirectionDefault {
		sortDirection = model.SortDirectionDesc
	}

	return builder.OrderBy(fmt.Sprintf("%s %s NULLS LAST", sortBy, sortDirection))
}

type User struct {
	ID            string
	Labels        map[string]interface{}
	CreatedAt     time.Time
	UpdatedAt     time.Time
	LastLoginAt   *time.Time
	IsDisabled    bool
	DisableReason *string
}

func (u *User) GetMeta() model.Meta {
	return model.Meta{
		ID:        u.ID,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

func (u *User) CheckStatus() error {
	if u.IsDisabled {
		return NewErrDisabledUser(u.DisableReason)
	}
	return nil
}

func newUserModel(
	user *User,
	identities []*identity.Info,
	isVerified bool,
) *model.User {
	isAnonymous := false
	for _, i := range identities {
		if i.Type == authn.IdentityTypeAnonymous {
			isAnonymous = true
			break
		}
	}

	return &model.User{
		Meta: model.Meta{
			ID:        user.ID,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
		},
		LastLoginAt:   user.LastLoginAt,
		IsAnonymous:   isAnonymous,
		IsVerified:    isVerified,
		IsDisabled:    user.IsDisabled,
		DisableReason: user.DisableReason,
	}
}
