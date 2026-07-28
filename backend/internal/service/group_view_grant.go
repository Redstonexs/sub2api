package service

import (
	"context"
	"time"
)

// GroupViewGrant 分组额度卡片查看授权：管理员授予普通用户查看某分组额度卡片的权限。
// 与 user_allowed_groups（API Key 绑定授权）语义完全独立，互不影响。
type GroupViewGrant struct {
	ID              int64
	UserID          int64
	GroupID         int64
	GrantedByUserID int64
	GrantedAt       time.Time
}

// GroupViewGrantWithUser 带用户名的授权记录（管理员在分组管理页查看授权列表）。
type GroupViewGrantWithUser struct {
	GroupViewGrant
	Username          string // 被授权用户
	Email             string // 被授权用户邮箱（用户名可能重复，邮箱用于消歧）
	GrantedByUsername string // 执行授权的管理员
}

// ViewableGroup 普通用户可查看额度卡片的分组。
type ViewableGroup struct {
	GroupID   int64
	GroupName string
	Platform  string
	Status    string
}

// GroupViewGrantRepository 授权记录持久化契约（由 repository 层实现，软删除感知）。
type GroupViewGrantRepository interface {
	Create(ctx context.Context, grant *GroupViewGrant) error
	DeleteByUserAndGroup(ctx context.Context, userID, groupID int64) error
	ListByGroup(ctx context.Context, groupID int64) ([]GroupViewGrantWithUser, error)
	ListByUser(ctx context.Context, userID int64) ([]ViewableGroup, error)
	ExistsByUserAndGroup(ctx context.Context, userID, groupID int64) (bool, error)
}
