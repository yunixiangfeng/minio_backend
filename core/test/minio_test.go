package test

import (
	"bytes"
	"context"
	"io"
	"log"
	"os"
	"testing"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/encrypt"
)

/**
测试MinIO对象存储服务的sdk
*/
func TestMinIO(t *testing.T) {
	ctx := context.Background()
	endpoint := "192.168.204.130:9000"
	accessKeyID := "minio123"
	secretAccessKey := "minio123"
	useSSL := false

	// Initialize minio client object.
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		log.Fatalln(err)
	}

	// Make a new bucket called mymusic.
	bucketName := "mymusic"
	location := "us-east-1"

	err = minioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{Region: location})
	if err != nil {
		// Check to see if we already own this bucket (which happens if you run this twice)
		exists, errBucketExists := minioClient.BucketExists(ctx, bucketName)
		if errBucketExists == nil && exists {
			log.Printf("We already own %s\n", bucketName)
		} else {
			log.Fatalln(err)
		}
	} else {
		log.Printf("Successfully created %s\n", bucketName)
	}

	// Upload the zip file
	objectName := "百事可乐的视频1"
	filePath := "img/百事可乐创意广告恶搞伦敦路人.mp4"
	contentType := "video/mpeg4"

	// Upload the zip file with FPutObject
	n, err := minioClient.FPutObject(ctx, bucketName, objectName, filePath, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		log.Fatalln(err)
	}
	log.Println(n)

	log.Printf("Successfully uploaded %s\n", objectName)
}

const (
	serverEndpoint = "192.168.204.130:9000"
	accessKey      = "minio123"
	secretKey      = "minio123"
	enableSecurity = false
)

func TestCoreMultipartUpload(t *testing.T) {
	if os.Getenv(serverEndpoint) == "" {
		t.Skip("SERVER_ENDPOINT not set")
	}
	if testing.Short() {
		t.Skip("skipping functional tests for the short runs")
	}

	// Instantiate new minio client object.
	core, err := minio.NewCore(
		os.Getenv(serverEndpoint),
		&minio.Options{
			Creds:  credentials.NewStaticV4(os.Getenv(accessKey), os.Getenv(secretKey), ""),
			Secure: enableSecurity,
		})
	if err != nil {
		t.Fatal("Error:", err)
	}

	bucketName := "wttest"
	// Make a new bucket.
	err = core.MakeBucket(context.Background(), bucketName, minio.MakeBucketOptions{Region: "us-east-1"})
	if err != nil {
		t.Fatal("Error:", err, bucketName)
	}
	objectName := "手撕golangv1.1.pdf"

	objectContentType := "binary/octet-stream"
	metadata := make(map[string]string)
	metadata["Content-Type"] = objectContentType
	putopts := minio.PutObjectOptions{
		UserMetadata: metadata,
	}
	uploadID, err := core.NewMultipartUpload(context.Background(), bucketName, objectName, putopts)
	if err != nil {
		t.Fatal("Error:", err, bucketName, objectName)
	}
	buf := bytes.Repeat([]byte("a"), 32*1024*1024)
	r := bytes.NewReader(buf)
	partBuf := make([]byte, 100*1024*1024)
	parts := make([]minio.CompletePart, 0, 5)
	partID := 0
	for {
		n, err := r.Read(partBuf)
		if err != nil && err != io.EOF {
			t.Fatal("Error:", err)
		}
		if err == io.EOF {
			break
		}
		if n > 0 {
			partID++
			data := bytes.NewReader(partBuf[:n])
			dataLen := int64(len(partBuf[:n]))
			objectPart, err := core.PutObjectPart(context.Background(), bucketName, objectName, uploadID, partID,
				data, dataLen,
				minio.PutObjectPartOptions{
					Md5Base64:    "",
					Sha256Hex:    "",
					SSE:          encrypt.NewSSE(),
					CustomHeader: nil,
					Trailer:      nil,
				},
			)
			if err != nil {
				t.Fatal("Error:", err, bucketName, objectName)
			}
			parts = append(parts, minio.CompletePart{
				PartNumber: partID,
				ETag:       objectPart.ETag,
			})
		}
	}

	// objectParts, err := core.listObjectParts(context.Background(), bucketName, objectName, uploadID)
	// if err != nil {
	// 	t.Fatal("Error:", err)
	// }
	// if len(objectParts) != len(parts) {
	// 	t.Fatal("Error", len(objectParts), len(parts))
	// }
	_, err = core.CompleteMultipartUpload(context.Background(), bucketName, objectName, uploadID, parts, putopts)
	if err != nil {
		t.Fatal("Error:", err)
	}

	if err := core.RemoveObject(context.Background(), bucketName, objectName, minio.RemoveObjectOptions{}); err != nil {
		t.Fatal("Error: ", err)
	}

	if err := core.RemoveBucket(context.Background(), bucketName); err != nil {
		t.Fatal("Error: ", err)
	}
}
