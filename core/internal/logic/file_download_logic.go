package logic

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"core/define"
	"core/internal/svc"
	"core/internal/types"
	"core/models"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/zeromicro/go-zero/core/logx"
)

type FileDownloadLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewFileDownloadLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FileDownloadLogic {
	return &FileDownloadLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// getMinioClient 获取 MinIO 客户端
func (l *FileDownloadLogic) getMinioClient() (*minio.Client, error) {
	return minio.New(define.MinIOEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(define.MinIOAccessKeyID, define.MinIOAccessSecretKey, ""),
		Secure: define.MinIOSSLBool,
	})
}

// queryRepositoryPool 通过 user_repository identity 查询 repository_pool 的 path
func (l *FileDownloadLogic) queryRepositoryPool(userRepoIdentity string) (*models.RepositoryPool, error) {
	ur := new(models.UserRepository)
	has, err := l.svcCtx.Engine.Where("identity = ?", userRepoIdentity).Get(ur)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, fmt.Errorf("文件记录不存在")
	}
	if ur.RepositoryIdentity == "" {
		return nil, fmt.Errorf("该条目不是文件，无法下载")
	}
	rp := new(models.RepositoryPool)
	has, err = l.svcCtx.Engine.Where("identity = ?", ur.RepositoryIdentity).Get(rp)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, fmt.Errorf("文件资源不存在")
	}
	return rp, nil
}

// FileDownload 单文件下载（流式输出）
func (l *FileDownloadLogic) FileDownload(req *types.FileDownloadRequest, w http.ResponseWriter, userIdentity string) error {
	rp, err := l.queryRepositoryPool(req.Identity)
	if err != nil {
		return err
	}

	client, err := l.getMinioClient()
	if err != nil {
		return fmt.Errorf("MinIO连接失败: %v", err)
	}

	// 从 path 字段解析 bucket 和 object name
	// path 格式: "bucketName/objectName" 如 "wttest/breakpoint/a/b/uuid.ext"
	bucketName, objectName, found := strings.Cut(rp.Path, "/")
	if !found {
		bucketName = define.MinIOBucket
		objectName = rp.Path
	}

	obj, err := client.GetObject(l.ctx, bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		return fmt.Errorf("获取文件失败: %v", err)
	}
	defer obj.Close()

	// 获取文件信息用于 Content-Disposition
	objInfo, err := obj.Stat()
	if err != nil {
		return fmt.Errorf("获取文件信息失败: %v", err)
	}

	filename := rp.Name
	if rp.Ext != "" && !strings.HasSuffix(rp.Name, rp.Ext) {
		filename = rp.Name + "." + rp.Ext
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", objInfo.Size))

	if _, err := io.Copy(w, obj); err != nil {
		return fmt.Errorf("传输文件失败: %v", err)
	}
	return nil
}

// FileBatchDownload 批量文件打包下载（zip）
func (l *FileDownloadLogic) FileBatchDownload(req *types.FileBatchDownloadRequest, w http.ResponseWriter, userIdentity string) error {
	if len(req.Identities) == 0 {
		return fmt.Errorf("请选择要下载的文件")
	}

	client, err := l.getMinioClient()
	if err != nil {
		return fmt.Errorf("MinIO连接失败: %v", err)
	}

	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	for _, ident := range req.Identities {
		rp, err := l.queryRepositoryPool(ident)
		if err != nil {
			logx.Errorf("查询文件 %s 失败: %v", ident, err)
			continue // 跳过无法下载的文件
		}

		bucketName, objectName, found := strings.Cut(rp.Path, "/")
		if !found {
			bucketName = define.MinIOBucket
			objectName = rp.Path
		}

		obj, err := client.GetObject(l.ctx, bucketName, objectName, minio.GetObjectOptions{})
		if err != nil {
			logx.Errorf("从 MinIO 获取文件 %s 失败: %v", objectName, err)
			continue
		}

		dlName := rp.Name
		if rp.Ext != "" && !strings.HasSuffix(rp.Name, rp.Ext) {
			dlName = rp.Name + "." + rp.Ext
		}

		f, err := zipWriter.Create(dlName)
		if err != nil {
			obj.Close()
			logx.Errorf("创建zip条目失败: %v", err)
			continue
		}

		if _, err := io.Copy(f, obj); err != nil {
			obj.Close()
			logx.Errorf("写入zip失败: %v", err)
			continue
		}
		obj.Close()
	}

	if err := zipWriter.Close(); err != nil {
		return fmt.Errorf("关闭zip失败: %v", err)
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="download.zip"`)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", buf.Len()))
	if _, err := w.Write(buf.Bytes()); err != nil {
		return fmt.Errorf("写入响应失败: %v", err)
	}
	return nil
}

// FileFolderDownload 文件夹打包下载（递归获取所有子文件并zip）
func (l *FileDownloadLogic) FileFolderDownload(req *types.FileFolderDownloadRequest, w http.ResponseWriter, userIdentity string) error {
	client, err := l.getMinioClient()
	if err != nil {
		return fmt.Errorf("MinIO连接失败: %v", err)
	}

	// 递归收集文件夹下所有文件
	var allFiles []fileEntry
	err = l.collectFilesRecursively(req.Identity, userIdentity, "", &allFiles)
	if err != nil {
		return err
	}
	if len(allFiles) == 0 {
		return fmt.Errorf("文件夹为空或无文件可下载")
	}

	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	for _, fe := range allFiles {
		bucketName, objectName, found := strings.Cut(fe.Path, "/")
		if !found {
			bucketName = define.MinIOBucket
			objectName = fe.Path
		}

		obj, err := client.GetObject(l.ctx, bucketName, objectName, minio.GetObjectOptions{})
		if err != nil {
			logx.Errorf("从 MinIO 获取文件 %s 失败: %v", objectName, err)
			continue
		}

		f, err := zipWriter.Create(filepath.FromSlash(fe.ZipPath))
		if err != nil {
			obj.Close()
			continue
		}

		io.Copy(f, obj)
		obj.Close()
	}

	zipWriter.Close()

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="folder_download.zip"`)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", buf.Len()))
	w.Write(buf.Bytes())
	return nil
}

// fileEntry 用于收集待下载的文件信息
type fileEntry struct {
	Name    string // 原始文件名
	Ext     string
	Path    string // MinIO 路径
	ZipPath string // zip 内部路径
}

// collectFilesRecursively 递归收集文件夹下所有文件
func (l *FileDownloadLogic) collectFilesRecursively(folderIdentity, userIdentity, prefix string, result *[]fileEntry) error {
	// 先通过 identity 获取文件夹的 id
	parentUR := new(models.UserRepository)
	has, err := l.svcCtx.Engine.Where("identity = ? AND user_identity = ? AND (deleted_at IS NULL OR deleted_at = '0001-01-01 00:00:00')",
		folderIdentity, userIdentity).Get(parentUR)
	if err != nil {
		return err
	}
	if !has {
		return fmt.Errorf("文件夹不存在")
	}

	// 查询当前文件夹下的直接子项（通过 parent_id）
	var children []models.UserRepository
	err = l.svcCtx.Engine.Where("parent_id = ? AND user_identity = ? AND (deleted_at IS NULL OR deleted_at = '0001-01-01 00:00:00')",
		parentUR.Id, userIdentity).Find(&children)
	if err != nil {
		return err
	}

	for _, child := range children {
		// 如果是文件夹（ext 为空且 repository_identity 为空），递归
		if child.Ext == "" && child.RepositoryIdentity == "" {
			newPrefix := prefix + child.Name + "/"
			l.collectFilesRecursively(child.Identity, userIdentity, newPrefix, result)
		} else if child.RepositoryIdentity != "" {
			// 文件，查找 repository_pool 获取路径
			rp := new(models.RepositoryPool)
			has, err := l.svcCtx.Engine.Where("identity = ?", child.RepositoryIdentity).Get(rp)
			if err != nil || !has {
				continue
			}
			filename := child.Name
			if child.Ext != "" && !strings.HasSuffix(child.Name, child.Ext) {
				filename = child.Name + "." + child.Ext
			}
			*result = append(*result, fileEntry{
				Name:    filename,
				Path:    rp.Path,
				ZipPath: prefix + filename,
			})
		}
	}
	return nil
}
