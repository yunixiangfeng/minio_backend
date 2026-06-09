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

	// 合并分片（key 即为 MinioInitPart 返回的 objectName，完整路径）
	if err = helper.MinioPartUploadComplete(req.Key, req.UploadId, mo); err != nil {
		return nil, err
	}

	// 数据入库，Path = bucket/objectName，与普通上传格式一致
	rp := &models.RepositoryPool{
		Identity: helper.UUID(),
		Hash:     req.Md5,
		Name:     req.Name,
		Ext:      req.Ext,
		Size:     req.Size,
		Path:     define.MinIOBucket + "/" + req.Key,
	}
	if _, err = l.svcCtx.Engine.Insert(rp); err != nil {
		return nil, err
	}

	resp = &types.FileUploadChunkCompleteReply{
		Identity: rp.Identity,
	}
	return
}
