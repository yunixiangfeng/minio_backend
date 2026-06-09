package handler

import (
	"crypto/md5"
	"fmt"
	"net/http"
	"path"

	"core/helper"
	"core/internal/logic"
	"core/internal/svc"
	"core/internal/types"
	"core/models"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func FileUploadHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.FileUploadRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		// 获取上传的文件（FormData）
		file, fileHeader, err := r.FormFile("file")
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, fmt.Errorf("获取上传文件失败: %v", err))
			return
		}
		defer file.Close()

		// 计算文件 MD5
		b := make([]byte, fileHeader.Size)
		if _, err = file.Read(b); err != nil {
			httpx.ErrorCtx(r.Context(), w, fmt.Errorf("读取文件内容失败: %v", err))
			return
		}
		hash := fmt.Sprintf("%x", md5.Sum(b))

		// 判断文件在数据库中是否已经存在（秒传）
		rp := new(models.RepositoryPool)
		has, err := svcCtx.Engine.Where("hash = ?", hash).Get(rp)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		if has {
			// 秒传：文件已存在，直接返回 identity
			httpx.OkJsonCtx(r.Context(), w, &types.FileUploadReply{
				Identity: rp.Identity,
				Ext:      rp.Ext,
				Name:     rp.Name,
			})
			return
		}

		// 往 minio 中存储文件（需要重置文件读取位置）
		// 由于已经 Read 了全部内容，需要重新 Seek 或重新调用 FormFile
		// 这里通过直接传 bytes 的方式重写 MinIOUpload
		filePath, err := helper.MinIOUploadBytes(b, fileHeader.Filename, fileHeader.Size)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, fmt.Errorf("MinIO上传失败: %v", err))
			return
		}

		// 往 logic 传递 request
		req.Name = fileHeader.Filename
		req.Ext = path.Ext(fileHeader.Filename)
		req.Size = fileHeader.Size
		req.Hash = hash
		req.Path = filePath

		l := logic.NewFileUploadLogic(r.Context(), svcCtx)
		resp, err := l.FileUpload(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
