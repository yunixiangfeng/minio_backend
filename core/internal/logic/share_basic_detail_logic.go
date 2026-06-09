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

// shareDetailRow 接收原始 SQL 查询结果（避免 XORM tag 干扰）
type shareDetailRow struct {
	RepositoryIdentity string    `xorm:"repository_identity"`
	ExpiredTime        int       `xorm:"expired_time"`
	CreatedAt          time.Time `xorm:"share_created_at"` // 使用别名避免与其他表字段冲突
	Name               string    `xorm:"name"`
	Ext                string    `xorm:"ext"`
	Size               int64     `xorm:"size"`
	Path               string    `xorm:"path"`
}

func (l *ShareBasicDetailLogic) ShareBasicDetail(req *types.ShareBasicDetailRequest) (resp *types.ShareBasicDetailReply, err error) {
	// 使用原生 SQL，明确指定 SELECT 别名，避免 XORM 多表 JOIN 字段冲突
	sqlStr := `
		SELECT
			sb.repository_identity,
			sb.expired_time,
			sb.created_at AS share_created_at,
			ur.name,
			rp.ext,
			rp.size,
			rp.path
		FROM share_basic sb
		LEFT JOIN repository_pool rp ON sb.repository_identity = rp.identity
		LEFT JOIN user_repository ur ON ur.identity = sb.user_repository_identity
		WHERE sb.identity = ?
		  AND (sb.deleted_at IS NULL OR sb.deleted_at = '0001-01-01 00:00:00')
		LIMIT 1
	`

	result := new(shareDetailRow)
	has, err := l.svcCtx.Engine.SQL(sqlStr, req.Identity).Get(result)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, errors.New("分享不存在或已过期")
	}

	// 判断过期（expired_time 为秒数，从 created_at 开始计算）
	if result.ExpiredTime > 0 {
		expireAt := result.CreatedAt.Add(time.Duration(result.ExpiredTime) * time.Second)
		if time.Now().After(expireAt) {
			return nil, errors.New("分享已过期")
		}
	}

	// 更新点击次数（忽略错误）
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
