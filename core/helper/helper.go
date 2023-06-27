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
// 邮箱验证码发送
func MailSendCode(mail, code string) error {
	e := email.NewEmail()
	e.From = "云盘 <535504958@qq.com>"
	e.To = []string{mail}
	e.Subject = "验证码发送测试"
	e.HTML = []byte(fmt.Sprintf("<pre style=\"font-family:Helvetica,arial,sans-serif;font-size:13px;color:#747474;text-align:left;line-height:18px\">欢迎使用水牛云盘，您的验证码为：<span style=\"font-size:block\">%s</span></pre>", code))
	err := e.SendWithTLS("smtp.163.com:465", smtp.PlainAuth("", "535504958@qq.com", define.MailPassword, "smtp.163.com"),
		&tls.Config{InsecureSkipVerify: true, ServerName: "smtp.163.com"})
	if err != nil {
		log.Print(err)
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

// MinIOUpload 上传到自建的minio中
func MinIOUpload(r *http.Request) (string, error) {
	minioClient, err := minio.New(define.MinIOEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(define.MinIOAccessKeyID, define.MinIOAccessSecretKey, ""),
		Secure: define.MinIOSSLBool,
	})
	if err != nil {
		return "", err
	}

	// // 获取文件信息
	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		return "", err
	}
	objectName := UUID() + path.Ext(fileHeader.Filename)

	_, err = minioClient.PutObject(context.Background(), define.MinIOBucket, objectName, file, fileHeader.Size,
		minio.PutObjectOptions{ContentType: "binary/octet-stream"})
	if err != nil {
		return "", err
	}
	return define.MinIOBucket + "/" + objectName, nil

}

// Minio 分片上传初始化
func MinioInitPart(ext string) (string, string, error) {
	// Instantiate new minio client object.
	core, err := minio.NewCore(
		define.MinIOEndpoint,
		&minio.Options{
			Creds:  credentials.NewStaticV4(define.MinIOAccessKeyID, define.MinIOAccessSecretKey, ""),
			Secure: define.MinIOSSLBool,
		})
	if err != nil {
		return "", "", err
	}
	key := "breakpoint/" + UUID() + ext
	uuid := uuid.NewV4().String()
	bucketName := define.MinIOBucket
	objectName := strings.TrimPrefix(path.Join(define.MINIO_BASE_PATH, path.Join(uuid[0:1], uuid[1:2], uuid)), "/")

	// objectContentType := "binary/octet-stream"
	// metadata := make(map[string]string)
	// metadata["Content-Type"] = objectContentType
	// putopts := minio.PutObjectOptions{
	// 	UserMetadata: metadata,
	// }
	uploadID, err := core.NewMultipartUpload(context.Background(), bucketName, objectName, minio.PutObjectOptions{})
	if err != nil {
		return "", "", err
	}
	return key, uploadID, nil
	// return core.NewMultipartUpload(bucketName, objectName, minio.PutObjectOptions{})
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

// // 分片上传完成
func MinioPartUploadComplete(key string, uploadID string, mo []minio.CompletePart) error {
	// Instantiate new minio client object.
	core, err := minio.NewCore(
		define.MinIOEndpoint,
		&minio.Options{
			Creds:  credentials.NewStaticV4(define.MinIOAccessKeyID, define.MinIOAccessSecretKey, ""),
			Secure: define.MinIOSSLBool,
		})
	if err != nil {
		return err
	}
	uuid := uuid.NewV4().String()
	bucketName := define.MinIOBucket
	objectName := strings.TrimPrefix(path.Join(define.MINIO_BASE_PATH, path.Join(uuid[0:1], uuid[1:2], uuid)), "/")
	// var client *minio_ext.Client
	// partInfos, err := client.ListObjectParts(bucketName, objectName, uploadID)
	// if err != nil {
	// 	return err
	// }

	var complMultipartUpload completeMultipartUpload
	// for _, partInfo := range partInfos {
	// 	complMultipartUpload.Parts = append(complMultipartUpload.Parts, minio.CompletePart{
	// 		PartNumber: partInfo.PartNumber,
	// 		ETag:       partInfo.ETag,
	// 	})

	// }
	complMultipartUpload.Parts = append(complMultipartUpload.Parts, mo...)

	objectContentType := "binary/octet-stream"
	metadata := make(map[string]string)
	metadata["Content-Type"] = objectContentType
	putopts := minio.PutObjectOptions{
		UserMetadata: metadata,
	}
	// Sort all completed parts.
	sort.Sort(completedParts(complMultipartUpload.Parts))
	_, err = core.CompleteMultipartUpload(context.Background(), bucketName, objectName, uploadID, complMultipartUpload.Parts, putopts)
	if err != nil {
		return err
	}

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
