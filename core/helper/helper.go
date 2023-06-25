package helper

import (
	"context"
	"core/define"
	"crypto/md5"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/smtp"
	"path"
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
		Creds: credentials.NewStaticV4(define.MinIOAccessKeyID, define.MinIOAccessSecretKey, ""),
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
