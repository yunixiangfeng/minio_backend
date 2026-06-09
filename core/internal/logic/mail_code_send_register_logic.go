package logic

import (
	"context"
	"errors"
	"log"
	"time"

	"core/define"
	"core/helper"
	"core/internal/svc"
	"core/internal/types"
	"core/models"

	"github.com/zeromicro/go-zero/core/logx"
)

type MailCodeSendRegisterLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewMailCodeSendRegisterLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MailCodeSendRegisterLogic {
	return &MailCodeSendRegisterLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *MailCodeSendRegisterLogic) MailCodeSendRegister(req *types.MailCodeSendRequest) (resp *types.MailCodeSendReply, err error) {
	// 该邮箱未被注册
	cnt, err := l.svcCtx.Engine.Where("email = ?", req.Email).Count(new(models.UserBasic))
	if err != nil {
		return
	}
	if cnt > 0 {
		err = errors.New("该邮箱已被注册")
		return
	}

	// 生成验证码
	code := helper.RandCode()
	log.Printf("[MailCodeSendRegister] email=%s code=%s", req.Email, code)

	// 先将验证码存入 Redis（必须成功，否则注册流程无法继续）
	if redisErr := l.svcCtx.RDB.Set(l.ctx, req.Email, code, time.Second*time.Duration(define.CodeExpire)).Err(); redisErr != nil {
		log.Printf("[MailCodeSendRegister] Redis Set error: %v", redisErr)
		return nil, errors.New("服务器内部错误，验证码存储失败，请联系管理员")
	}

	// 尝试发送邮件
	if mailErr := helper.MailSendCode(req.Email, code); mailErr != nil {
		log.Printf("[MailCodeSendRegister] 邮件发送失败，验证码已存入Redis: email=%s code=%s err=%v", req.Email, code, mailErr)
		// 邮件失败时把验证码直接返回给前端（便于测试；生产环境可以删除 Code 字段）
		return &types.MailCodeSendReply{Code: code}, nil
	}

	// 邮件成功，不返回验证码（前端从邮箱查收）
	return &types.MailCodeSendReply{}, nil
}
