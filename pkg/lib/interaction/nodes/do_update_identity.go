package nodes

import (
	"errors"

	"github.com/authgear/authgear-server/pkg/api/event"
	"github.com/authgear/authgear-server/pkg/lib/authn/identity"
	"github.com/authgear/authgear-server/pkg/lib/interaction"
)

func init() {
	interaction.RegisterNode(&NodeDoUpdateIdentity{})
}

type EdgeDoUpdateIdentity struct {
	IdentityBeforeUpdate *identity.Info
	IdentityAfterUpdate  *identity.Info
}

func (e *EdgeDoUpdateIdentity) Instantiate(ctx *interaction.Context, graph *interaction.Graph, rawInput interface{}) (interaction.Node, error) {
	return &NodeDoUpdateIdentity{
		IdentityBeforeUpdate: e.IdentityBeforeUpdate,
		IdentityAfterUpdate:  e.IdentityAfterUpdate,
	}, nil
}

type NodeDoUpdateIdentity struct {
	IdentityBeforeUpdate *identity.Info `json:"identity_before_update"`
	IdentityAfterUpdate  *identity.Info `json:"identity_after_update"`
}

func (n *NodeDoUpdateIdentity) Prepare(ctx *interaction.Context, graph *interaction.Graph) error {
	return nil
}

func (n *NodeDoUpdateIdentity) GetEffects() ([]interaction.Effect, error) {
	return []interaction.Effect{
		interaction.EffectRun(func(ctx *interaction.Context, graph *interaction.Graph, nodeIndex int) error {
			if _, err := ctx.Identities.CheckDuplicated(n.IdentityAfterUpdate); err != nil {
				if errors.Is(err, identity.ErrIdentityAlreadyExists) {
					return interaction.ErrDuplicatedIdentity
				}
				return err
			}

			if err := ctx.Identities.Update(n.IdentityAfterUpdate); err != nil {
				return err
			}

			return nil
		}),
		interaction.EffectOnCommit(func(ctx *interaction.Context, graph *interaction.Graph, nodeIndex int) error {
			user, err := ctx.Users.Get(n.IdentityAfterUpdate.UserID)
			if err != nil {
				return err
			}

			err = ctx.Hooks.DispatchEvent(&event.IdentityUpdateEvent{
				User:        *user,
				OldIdentity: n.IdentityBeforeUpdate.ToModel(),
				NewIdentity: n.IdentityAfterUpdate.ToModel(),
			})
			if err != nil {
				return err
			}

			return nil
		}),
	}, nil
}

func (n *NodeDoUpdateIdentity) DeriveEdges(graph *interaction.Graph) ([]interaction.Edge, error) {
	return graph.Intent.DeriveEdgesForNode(graph, n)
}

func (n *NodeDoUpdateIdentity) UserIdentity() *identity.Info {
	return n.IdentityAfterUpdate
}
