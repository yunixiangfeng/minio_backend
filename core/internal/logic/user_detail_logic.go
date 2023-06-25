package logic

import (
	"context"
	"errors"

	"core/internal/svc"
	"core/internal/types"
	"core/models"

	"github.com/zeromicro/go-zero/core/logx"
)

type UserDetailLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUserDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UserDetailLogic {
	return &UserDetailLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UserDetailLogic) UserDetail(req *types.UserDetailRequest) (resp *types.UserDetailReply, err error) {
	ub := new(models.UserBasic)
	has, err := l.svcCtx.Engine.Where("identity = ?", req.Identity).Get(ub)
	if err != nil {
		return
	}
	// 用户不存在
	if !has {
		return nil, errors.New("user not found")
	}

	resp = new(types.UserDetailReply)
	resp.Name = ub.Name
	resp.Email = ub.Email

	return
}
