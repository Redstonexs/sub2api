package service

import (
	"context"
	"fmt"
	"time"
)

// GroupViewGrantService 分组额度卡片查看授权服务。
// 管理员通过 GrantViewAccess 授予普通用户查看某分组额度卡片的权限；
// 通过 RevokeViewAccess 撤销授权；ListViewableGroupsByUser 返回用户可查看的
// 分组列表（仅限 Anthropic/OpenAI 平台且激活状态的分组）。
type GroupViewGrantService struct {
	grants GroupViewGrantRepository
}

// NewGroupViewGrantService 创建 GroupViewGrantService 实例。
func NewGroupViewGrantService(grants GroupViewGrantRepository) *GroupViewGrantService {
	return &GroupViewGrantService{grants: grants}
}

// GrantViewAccess 授权普通用户查看指定分组的额度卡片。
// 幂等：若已有活跃授权则直接返回 nil，不重复创建。
func (s *GroupViewGrantService) GrantViewAccess(
	ctx context.Context, groupID, userID, grantedByUserID int64,
) error {
	exists, err := s.grants.ExistsByUserAndGroup(ctx, userID, groupID)
	if err != nil {
		return fmt.Errorf("check existing grant: %w", err)
	}
	if exists {
		return nil
	}

	grant := &GroupViewGrant{
		UserID:          userID,
		GroupID:         groupID,
		GrantedByUserID: grantedByUserID,
		GrantedAt:       time.Now(),
	}
	if err := s.grants.Create(ctx, grant); err != nil {
		return fmt.Errorf("create grant: %w", err)
	}
	return nil
}

// RevokeViewAccess 撤销普通用户对指定分组额度卡片的查看权限。
// 幂等：若无活跃授权则直接返回 nil。
func (s *GroupViewGrantService) RevokeViewAccess(
	ctx context.Context, groupID, userID int64,
) error {
	exists, err := s.grants.ExistsByUserAndGroup(ctx, userID, groupID)
	if err != nil {
		return fmt.Errorf("check existing grant: %w", err)
	}
	if !exists {
		return nil
	}

	if err := s.grants.DeleteByUserAndGroup(ctx, userID, groupID); err != nil {
		return fmt.Errorf("delete grant: %w", err)
	}
	return nil
}

// ListGrantsByGroup 查询指定分组的所有授权记录（含用户名）。
func (s *GroupViewGrantService) ListGrantsByGroup(
	ctx context.Context, groupID int64,
) ([]GroupViewGrantWithUser, error) {
	grants, err := s.grants.ListByGroup(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("list grants by group: %w", err)
	}
	return grants, nil
}

// ListViewableGroupsByUser 返回指定用户可查看额度卡片的分组列表。
// 仅保留 Anthropic 或 OpenAI 平台且状态为激活的分组。
func (s *GroupViewGrantService) ListViewableGroupsByUser(
	ctx context.Context, userID int64,
) ([]ViewableGroup, error) {
	groups, err := s.grants.ListByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list groups by user: %w", err)
	}

	filtered := make([]ViewableGroup, 0, len(groups))
	for _, g := range groups {
		switch g.Platform {
		case PlatformAnthropic, PlatformOpenAI:
			if g.Status == StatusActive {
				filtered = append(filtered, g)
			}
		case PlatformGemini, PlatformAntigravity, PlatformGrok, PlatformComposite:
			// 不支持这些平台的分组额度卡片查看
		default:
			// 未知平台，跳过
		}
	}
	return filtered, nil
}

// HasViewAccess 判断指定用户是否有权查看指定分组的额度卡片。
func (s *GroupViewGrantService) HasViewAccess(
	ctx context.Context, userID, groupID int64,
) (bool, error) {
	ok, err := s.grants.ExistsByUserAndGroup(ctx, userID, groupID)
	if err != nil {
		return false, fmt.Errorf("check grant exists: %w", err)
	}
	return ok, nil
}
