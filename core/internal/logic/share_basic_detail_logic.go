package logic

import (
	"context"
	"errors"
	"time"

	"core/internal/svc"
	"core/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ShareBasicDetailLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewShareBasicDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ShareBasicDetailLogic {
	return &ShareBasicDetailLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// shareDetailResult 用于接收多表JOIN的查询结果
type shareDetailResult struct {
	RepositoryIdentity string `xorm:"repository_identity"`
	Name               string `xorm:"name"`
	Ext                string `xorm:"ext"`
	Size               int64  `xorm:"size"`
	Path               string `xorm:"path"`
	ExpiredTime        int    `xorm:"expired_time"`
	CreatedAt          time.Time `xorm:"created_at"`
}

func (l *ShareBasicDetailLogic) ShareBasicDetail(req *types.ShareBasicDetailRequest) (resp *types.ShareBasicDetailReply, err error) {
	// 先查分享记录，判断是否存在及是否过期
	result := new(shareDetailResult)
	has, err := l.svcCtx.Engine.Table("share_basic").
		Select("share_basic.repository_identity, share_basic.expired_time, share_basic.created_at, user_repository.name, repository_pool.ext, repository_pool.size, repository_pool.path").
		Join("LEFT", "repository_pool", "share_basic.repository_identity = repository_pool.identity").
		Join("LEFT", "user_repository", "user_repository.identity = share_basic.user_repository_identity").
		Where("share_basic.identity = ? AND (share_basic.deleted_at IS NULL OR share_basic.deleted_at = '0001-01-01 00:00:00')", req.Identity).
		Get(result)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, errors.New("分享不存在或已过期")
	}

	// 判断是否过期（expired_time 为秒数，从 created_at 开始计算）
	if result.ExpiredTime > 0 {
		expireAt := result.CreatedAt.Add(time.Duration(result.ExpiredTime) * time.Second)
		if time.Now().After(expireAt) {
			return nil, errors.New("分享已过期")
		}
	}

	// 更新点击次数（不影响主流程）
	l.svcCtx.Engine.Exec("UPDATE share_basic SET click_num = click_num + 1 WHERE identity = ?", req.Identity)

	resp = &types.ShareBasicDetailReply{
		RepositoryIdentity: result.RepositoryIdentity,
		Name:               result.Name,
		Ext:                result.Ext,
		Size:               result.Size,
		Path:               result.Path,
	}
	return
}
