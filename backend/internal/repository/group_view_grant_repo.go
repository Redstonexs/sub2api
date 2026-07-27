package repository

import (
	"context"
	"errors"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/groupviewgrant"
	"github.com/Wei-Shaw/sub2api/ent/user"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type groupViewGrantRepository struct {
	client *dbent.Client
}

var _ service.GroupViewGrantRepository = (*groupViewGrantRepository)(nil)

// NewGroupViewGrantRepository 创建分组额度卡片查看授权仓储。
// 软删除由 SoftDeleteMixin 的拦截器/钩子处理：普通查询自动过滤 deleted_at，
// Delete 自动转为 UPDATE SET deleted_at = NOW()，无需显式处理。
func NewGroupViewGrantRepository(client *dbent.Client) service.GroupViewGrantRepository {
	return &groupViewGrantRepository{client: client}
}

func (r *groupViewGrantRepository) Create(ctx context.Context, grant *service.GroupViewGrant) error {
	if grant == nil {
		return errors.New("group view grant is nil")
	}
	client := clientFromContext(ctx, r.client)
	builder := client.GroupViewGrant.Create().
		SetUserID(grant.UserID).
		SetGroupID(grant.GroupID).
		SetGrantedByUserID(grant.GrantedByUserID)
	if !grant.GrantedAt.IsZero() {
		builder.SetGrantedAt(grant.GrantedAt)
	}
	created, err := builder.Save(ctx)
	if err != nil {
		return err
	}
	grant.ID = created.ID
	grant.GrantedAt = created.GrantedAt
	return nil
}

func (r *groupViewGrantRepository) DeleteByUserAndGroup(ctx context.Context, userID, groupID int64) error {
	client := clientFromContext(ctx, r.client)
	_, err := client.GroupViewGrant.Delete().
		Where(
			groupviewgrant.UserIDEQ(userID),
			groupviewgrant.GroupIDEQ(groupID),
		).
		Exec(ctx)
	return err
}

func (r *groupViewGrantRepository) ListByGroup(ctx context.Context, groupID int64) ([]service.GroupViewGrantWithUser, error) {
	client := clientFromContext(ctx, r.client)
	grants, err := client.GroupViewGrant.Query().
		Where(groupviewgrant.GroupIDEQ(groupID)).
		Order(dbent.Asc(groupviewgrant.FieldGrantedAt), dbent.Asc(groupviewgrant.FieldID)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	if len(grants) == 0 {
		return []service.GroupViewGrantWithUser{}, nil
	}

	// 批量加载被授权人与授权人的用户名（一次查询，避免 N+1）。
	userIDs := make([]int64, 0, len(grants)*2)
	seen := make(map[int64]struct{}, len(grants)*2)
	for _, g := range grants {
		for _, id := range [2]int64{g.UserID, g.GrantedByUserID} {
			if _, ok := seen[id]; !ok {
				seen[id] = struct{}{}
				userIDs = append(userIDs, id)
			}
		}
	}
	users, err := client.User.Query().
		Where(user.IDIn(userIDs...)).
		Select(user.FieldID, user.FieldUsername).
		All(ctx)
	if err != nil {
		return nil, err
	}
	usernames := make(map[int64]string, len(users))
	for _, u := range users {
		usernames[u.ID] = u.Username
	}

	out := make([]service.GroupViewGrantWithUser, 0, len(grants))
	for _, g := range grants {
		out = append(out, service.GroupViewGrantWithUser{
			GroupViewGrant: service.GroupViewGrant{
				ID:              g.ID,
				UserID:          g.UserID,
				GroupID:         g.GroupID,
				GrantedByUserID: g.GrantedByUserID,
				GrantedAt:       g.GrantedAt,
			},
			Username:          usernames[g.UserID],
			GrantedByUsername: usernames[g.GrantedByUserID],
		})
	}
	return out, nil
}

func (r *groupViewGrantRepository) ListByUser(ctx context.Context, userID int64) ([]service.ViewableGroup, error) {
	client := clientFromContext(ctx, r.client)
	grants, err := client.GroupViewGrant.Query().
		Where(groupviewgrant.UserIDEQ(userID)).
		WithGroup().
		Order(dbent.Asc(groupviewgrant.FieldGrantedAt), dbent.Asc(groupviewgrant.FieldID)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]service.ViewableGroup, 0, len(grants))
	for _, g := range grants {
		// 分组已软删除时 edge 加载为空，该分组不再可查看，跳过。
		grp := g.Edges.Group
		if grp == nil {
			continue
		}
		out = append(out, service.ViewableGroup{
			GroupID:   grp.ID,
			GroupName: grp.Name,
			Platform:  grp.Platform,
			Status:    grp.Status,
		})
	}
	return out, nil
}

func (r *groupViewGrantRepository) ExistsByUserAndGroup(ctx context.Context, userID, groupID int64) (bool, error) {
	client := clientFromContext(ctx, r.client)
	return client.GroupViewGrant.Query().
		Where(
			groupviewgrant.UserIDEQ(userID),
			groupviewgrant.GroupIDEQ(groupID),
		).
		Exist(ctx)
}
