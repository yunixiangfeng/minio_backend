package logic

import (
	"context"

	"core/define"
	"core/helper"
	"core/internal/svc"
	"core/internal/types"
	"core/models"

	"github.com/minio/minio-go/v7"
	"github.com/zeromicro/go-zero/core/logx"
)

type FileUploadChunkCompleteLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewFileUploadChunkCompleteLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FileUploadChunkCompleteLogic {
	return &FileUploadChunkCompleteLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *FileUploadChunkCompleteLogic) FileUploadChunkComplete(req *types.FileUploadChunkCompleteRequest) (resp *types.FileUploadChunkCompleteReply, err error) {
	mo := make([]minio.CompletePart, 0)
	for _, v := range req.MinioObjects {
		mo = append(mo, minio.CompletePart{
			ETag:       v.Etag,
			PartNumber: v.PartNumber,
		})
	}
	err = helper.MinioPartUploadComplete(req.Key, req.UploadId, mo)

	// 数据入库
	rp := &models.RepositoryPool{
		Identity: helper.UUID(),
		Hash:     req.Md5,
		Name:     req.Name,
		Ext:      req.Ext,
		Size:     req.Size,
		Path:     define.MinIOBucket + "/" + req.Key,
	}
	l.svcCtx.Engine.Insert(rp)
	return
}
