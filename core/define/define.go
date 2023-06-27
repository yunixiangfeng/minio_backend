package define

import (
	"github.com/dgrijalva/jwt-go"
)

type UserClaim struct {
	Id       int
	Identity string
	Name     string
	jwt.StandardClaims
}

var JwtKey = "minio-backend-key"
var MailPassword = "123456789qwertyu"

// CodeLength 验证码长度
var CodeLength = 6

// CodeExpire 验证码过期时间（s）
var CodeExpire = 300

// MinIOAccessKeyID MinIO 配置
var MinIOAccessKeyID = "minio123"
var MinIOAccessSecretKey = "minio123"
var MinIOEndpoint = "192.168.204.130:9000"
var MinIOBucket = "wttest"
var MinIOBucketLocation = "beijing"
var MinIOSSLBool = false
var MINIO_BASE_PATH = "breakpoint"

// BucketPolicy 设置存储桶权限
var MinIOBucketPolicy = "{\"Version\":\"2012-10-17\",\"Statement\":[{\"Effect\":\"Allow\",\"Principal\":{\"AWS\":[\"*\"]},\"Action\":[\"s3:GetBucketLocation\",\"s3:ListBucket\",\"s3:ListBucketMultipartUploads\"],\"Resource\":[\"arn:aws:s3:::%s\"]},{\"Effect\":\"Allow\",\"Principal\":{\"AWS\":[\"*\"]},\"Action\":[\"s3:AbortMultipartUpload\",\"s3:DeleteObject\",\"s3:GetObject\",\"s3:ListMultipartUploadParts\",\"s3:PutObject\"],\"Resource\":[\"arn:aws:s3:::%s/*\"]}]}"

// PageSize 分页的默认参数
var PageSize = 20

var Datetime = "2006-01-02 15:04:05"

var TokenExpire = 3600
var RefreshTokenExpire = 7200
