package logic

import (
	"context"

	"core/internal/svc"
	"core/internal/types"
	"core/models"

	"github.com/zeromicro/go-zero/core/logx"
)

type UserFolderListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUserFolderListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UserFolderListLogic {
	return &UserFolderListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// UserFolderList 返回当前用户的所有文件夹（用于移动文件时选择目标目录）
func (l *UserFolderListLogic) UserFolderList(req *types.UserFolderListRequest, userIdentity string) (resp *types.UserFolderListReply, err error) {
	resp = new(types.UserFolderListReply)
	resp.List = make([]*types.UserFolder, 0)

	var folders []models.UserRepository
	query := l.svcCtx.Engine.Where(
		"user_identity = ? AND ext = '' AND repository_identity = '' AND (deleted_at IS NULL OR deleted_at = '0001-01-01 00:00:00')",
		userIdentity,
	)
	// 如果指定了父目录 identity，则只查该目录下的文件夹
	if req.Identity != "" {
		// 先找父目录的 id
		parent := new(models.UserRepository)
		has, e := l.svcCtx.Engine.Where("identity = ? AND user_identity = ?", req.Identity, userIdentity).Get(parent)
		if e != nil {
			return nil, e
		}
		if has {
			query = query.And("parent_id = ?", parent.Id)
		}
	}

	if err = query.Find(&folders); err != nil {
		return nil, err
	}

	for _, f := range folders {
		resp.List = append(resp.List, &types.UserFolder{
			Identity: f.Identity,
			Name:     f.Name,
		})
	}
	return
}
