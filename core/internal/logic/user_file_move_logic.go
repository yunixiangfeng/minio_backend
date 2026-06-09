package logic

import (
	"context"
	"errors"

	"core/internal/svc"
	"core/internal/types"
	"core/models"

	"github.com/zeromicro/go-zero/core/logx"
)

type UserFileMoveLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUserFileMoveLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UserFileMoveLogic {
	return &UserFileMoveLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UserFileMoveLogic) UserFileMove(req *types.UserFileMoveRequest, userIdentity string) (resp *types.UserFileMoveReply, err error) {
	// 确认被移动的文件/文件夹存在
	fileData := new(models.UserRepository)
	has, err := l.svcCtx.Engine.Where("identity = ? AND user_identity = ?", req.Idnetity, userIdentity).Get(fileData)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, errors.New("文件不存在")
	}

	var newParentId int64 = 0 // 默认移到根目录

	// 如果指定了目标文件夹 identity，查找其 id
	if req.ParentIdnetity != "" {
		parentData := new(models.UserRepository)
		has, err = l.svcCtx.Engine.Where("identity = ? AND user_identity = ?", req.ParentIdnetity, userIdentity).Get(parentData)
		if err != nil {
			return nil, err
		}
		if !has {
			return nil, errors.New("目标文件夹不存在")
		}
		// 不能移动到自身
		if parentData.Identity == fileData.Identity {
			return nil, errors.New("不能移动到自身")
		}
		newParentId = int64(parentData.Id)
	}

	// 更新 parent_id
	_, err = l.svcCtx.Engine.Where("identity = ? AND user_identity = ?", req.Idnetity, userIdentity).
		Cols("parent_id").
		Update(&models.UserRepository{ParentId: newParentId})
	return
}
