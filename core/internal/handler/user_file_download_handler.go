package handler

import (
	"net/http"

	"core/internal/logic"
	"core/internal/svc"
	"core/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

// UserFileDownloadHandler 单文件下载
func UserFileDownloadHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.FileDownloadRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		l := logic.NewFileDownloadLogic(r.Context(), svcCtx)
		if err := l.FileDownload(&req, w, r.Header.Get("UserIdentity")); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		}
	}
}

// UserFileBatchDownloadHandler 批量文件打包下载
func UserFileBatchDownloadHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.FileBatchDownloadRequest
		if err := httpx.ParseJsonBody(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		l := logic.NewFileDownloadLogic(r.Context(), svcCtx)
		if err := l.FileBatchDownload(&req, w, r.Header.Get("UserIdentity")); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		}
	}
}

// UserFileFolderDownloadHandler 文件夹打包下载
func UserFileFolderDownloadHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.FileFolderDownloadRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		l := logic.NewFileDownloadLogic(r.Context(), svcCtx)
		if err := l.FileFolderDownload(&req, w, r.Header.Get("UserIdentity")); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		}
	}
}
