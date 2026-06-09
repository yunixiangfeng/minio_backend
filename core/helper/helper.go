package helper

import (
	"bytes"
	"context"
	"core/define"
	"crypto/md5"
	"crypto/tls"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/smtp"
	"net/url"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/jordan-wright/email"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	uuid "github.com/satori/go.uuid"
)

func Md5(s string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(s)))
}

func GenerateToken(id int, identity, name string, second int) (string, error) {
	uc := define.UserClaim{
		Id:       id,
		Identity: identity,
		Name:     name,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Second * time.Duration(second)).Unix(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, uc)
	tokenString, err := token.SignedString([]byte(define.JwtKey))
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

// AnalyzeToken
// Token 解析
func AnalyzeToken(token string) (*define.UserClaim, error) {
	uc := new(define.UserClaim)
	claims, err := jwt.ParseWithClaims(token, uc, func(token *jwt.Token) (interface{}, error) {
		return []byte(define.JwtKey), nil
	})
	if err != nil {
		return nil, err
	}
	if !claims.Valid {
		return uc, errors.New("token is invalid")
	}
	return uc, err
}

// MailSendCode
// 邮箱验证码发送（使用QQ邮箱 SMTP SSL）
func MailSendCode(mail, code string) error {
	e := email.NewEmail()
	e.From = "MinIO云盘 <535504958@qq.com>"
	e.To = []string{mail}
	e.Subject = "【MinIO云盘】您的验证码"
	e.HTML = []byte(fmt.Sprintf(
		"<div style=\"font-family:Helvetica,arial,sans-serif;font-size:14px;color:#333\">"+
			"<p>欢迎使用 MinIO 云盘！</p>"+
			"<p>您的注册验证码为：<strong style=\"font-size:20px;color:#409eff\">%s</strong></p>"+
			"<p style=\"color:#999;font-size:12px\">验证码 %d 分钟内有效，请勿泄露给他人。</p>"+
			"</div>",
		code, define.CodeExpire/60,
	))
	// QQ邮箱 SMTP over SSL，端口 465
	err := e.SendWithTLS(
		"smtp.qq.com:465",
		smtp.PlainAuth("", "535504958@qq.com", define.MailPassword, "smtp.qq.com"),
		&tls.Config{InsecureSkipVerify: true, ServerName: "smtp.qq.com"},
	)
	if err != nil {
		log.Print("MailSendCode error:", err)
		return err
	}
	return nil
}

func RandCode() string {
	s := "1234567890"
	code := ""
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < define.CodeLength; i++ {
		code += string(s[rand.Intn(len(s))])
	}
	return code
}

func UUID() string {
	return uuid.NewV4().String()
}

// MinIOUpload 上传到自建的minio中（原版，保留兼容）
func MinIOUpload(r *http.Request) (string, error) {
	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		return "", err
	}
	defer file.Close()
	b := make([]byte, fileHeader.Size)
	if _, err = file.Read(b); err != nil {
		return "", err
	}
	return MinIOUploadBytes(b, fileHeader.Filename, fileHeader.Size)
}

// MinIOUploadBytes 上传字节数组到 MinIO（避免 http.Request 文件指针二次读取问题）
func MinIOUploadBytes(data []byte, filename string, size int64) (string, error) {
	minioClient, err := minio.New(define.MinIOEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(define.MinIOAccessKeyID, define.MinIOAccessSecretKey, ""),
		Secure: define.MinIOSSLBool,
	})
	if err != nil {
		return "", err
	}

	// 确保 bucket 存在
	ctx := context.Background()
	exists, err := minioClient.BucketExists(ctx, define.MinIOBucket)
	if err != nil {
		return "", fmt.Errorf("检查bucket失败: %v", err)
	}
	if !exists {
		if err = minioClient.MakeBucket(ctx, define.MinIOBucket, minio.MakeBucketOptions{Region: define.MinIOBucketLocation}); err != nil {
			return "", fmt.Errorf("创建bucket失败: %v", err)
		}
	}

	objectName := UUID() + path.Ext(filename)
	reader := bytes.NewReader(data)
	_, err = minioClient.PutObject(ctx, define.MinIOBucket, objectName, reader, size,
		minio.PutObjectOptions{ContentType: "binary/octet-stream"})
	if err != nil {
		return "", fmt.Errorf("PutObject失败: %v", err)
	}
	return define.MinIOBucket + "/" + objectName, nil
}

// Minio 分片上传初始化
// 返回 objectName（即 key）和 uploadID，两者一致
func MinioInitPart(ext string) (string, string, error) {
	core, err := minio.NewCore(
		define.MinIOEndpoint,
		&minio.Options{
			Creds:  credentials.NewStaticV4(define.MinIOAccessKeyID, define.MinIOAccessSecretKey, ""),
			Secure: define.MinIOSSLBool,
		})
	if err != nil {
		return "", "", err
	}

	// 确保 bucket 存在
	ctx := context.Background()
	minioClient, _ := minio.New(define.MinIOEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(define.MinIOAccessKeyID, define.MinIOAccessSecretKey, ""),
		Secure: define.MinIOSSLBool,
	})
	if minioClient != nil {
		exists, _ := minioClient.BucketExists(ctx, define.MinIOBucket)
		if !exists {
			minioClient.MakeBucket(ctx, define.MinIOBucket, minio.MakeBucketOptions{Region: define.MinIOBucketLocation})
		}
	}

	// 生成唯一的对象路径，作为 MinIO objectName 和后续使用的 key
	id := uuid.NewV4().String()
	objectName := strings.TrimPrefix(
		path.Join(define.MINIO_BASE_PATH, path.Join(id[0:1], id[1:2], id)+ext),
		"/",
	)

	uploadID, err := core.NewMultipartUpload(ctx, define.MinIOBucket, objectName, minio.PutObjectOptions{})
	if err != nil {
		return "", "", err
	}
	// key 和 objectName 保持一致，前端传回 key 给 chunk 和 complete 时使用同一路径
	return objectName, uploadID, nil
}

// // 分片上传
func MinioPartUpload(r *http.Request) (string, error) {
	// Instantiate new minio client object.
	core, err := minio.NewCore(
		define.MinIOEndpoint,
		&minio.Options{
			Creds:  credentials.NewStaticV4(define.MinIOAccessKeyID, define.MinIOAccessSecretKey, ""),
			Secure: define.MinIOSSLBool,
		})
	if err != nil {
		return "", err
	}

	key := r.PostForm.Get("key")
	UploadID := r.PostForm.Get("upload_id")
	partNumber, err := strconv.Atoi(r.PostForm.Get("part_number"))

	if err != nil {
		return "", err
	}

	f, _, err := r.FormFile("file")
	if err != nil {
		return "", nil
	}

	buf := bytes.NewBuffer(nil)
	io.Copy(buf, f)

	data := bytes.NewReader(buf.Bytes())
	dataLen := int64(len(buf.Bytes()))
	// PutObjectPartOptions contains options for PutObjectPart API
	// type PutObjectPartOptions struct {
	// 	Md5Base64, Sha256Hex  string
	// 	SSE                   encrypt.ServerSide
	// 	CustomHeader, Trailer http.Header
	// }
	// opt可选
	objectPart, err := core.PutObjectPart(context.Background(), define.MinIOBucket, key, UploadID, partNumber, data, dataLen, minio.PutObjectPartOptions{})
	if err != nil {
		return "", err
	}

	return objectPart.ETag, nil
}

type completeMultipartUpload struct {
	XMLName xml.Name             `xml:"http://s3.amazonaws.com/doc/2006-03-01/ CompleteMultipartUpload" json:"-"`
	Parts   []minio.CompletePart `xml:"Part"`
}
type ComplPart struct {
	PartNumber int    `json:"partNumber"`
	ETag       string `json:"eTag"`
}

type completedParts []minio.CompletePart

func (a completedParts) Len() int           { return len(a) }
func (a completedParts) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a completedParts) Less(i, j int) bool { return a[i].PartNumber < a[j].PartNumber }

type CompleteParts struct {
	Data []ComplPart `json:"completedParts"`
}

// MinioPartUploadComplete 完成分片上传
// key 是 MinioInitPart 返回的 objectName，uploadID 是对应的上传 ID
func MinioPartUploadComplete(key string, uploadID string, mo []minio.CompletePart) error {
	core, err := minio.NewCore(
		define.MinIOEndpoint,
		&minio.Options{
			Creds:  credentials.NewStaticV4(define.MinIOAccessKeyID, define.MinIOAccessSecretKey, ""),
			Secure: define.MinIOSSLBool,
		})
	if err != nil {
		return err
	}

	var complMultipartUpload completeMultipartUpload
	complMultipartUpload.Parts = append(complMultipartUpload.Parts, mo...)

	// 按 PartNumber 排序
	sort.Sort(completedParts(complMultipartUpload.Parts))

	putopts := minio.PutObjectOptions{
		UserMetadata: map[string]string{"Content-Type": "binary/octet-stream"},
	}
	// 使用传入的 key（即 objectName），而非重新生成路径
	_, err = core.CompleteMultipartUpload(context.Background(), define.MinIOBucket, key, uploadID, complMultipartUpload.Parts, putopts)
	return err
}

func genMultiPartSignedUrl(uuid string, uploadId string, partNumber int, partSize int64) (*url.URL, error) {
	minioClient, err := minio.New(define.MinIOEndpoint, &minio.Options{
		Creds: credentials.NewStaticV4(define.MinIOAccessKeyID, define.MinIOAccessSecretKey, ""),
	})
	if err != nil {
		return nil, err
	}

	bucketName := define.MinIOBucket
	objectName := strings.TrimPrefix(path.Join(define.MINIO_BASE_PATH, path.Join(uuid[0:1], uuid[1:2], uuid)), "/")
	method := http.MethodPost
	expires := time.Hour * 24 * 7
	reqParams := make(url.Values)
	return minioClient.Presign(context.Background(), method, bucketName, objectName, expires, reqParams)
}
