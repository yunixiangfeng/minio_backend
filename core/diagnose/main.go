package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func main() {
	dsn := "root:1234@tcp(192.168.204.130:3306)/minio-backend?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal("db open:", err)
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		log.Fatal("db ping:", err)
	}
	fmt.Println("=== MySQL Connected ===")

	// 查 repository_pool
	rows, err := db.Query("SELECT id, identity, name, ext, path, hash FROM repository_pool LIMIT 20")
	if err != nil {
		log.Fatal("query repository_pool:", err)
	}
	defer rows.Close()
	fmt.Println("\n--- repository_pool ---")
	for rows.Next() {
		var id int
		var identity, name, ext, fpath, hash string
		rows.Scan(&id, &identity, &name, &ext, &fpath, &hash)
		fmt.Printf("  id=%d identity=%s name=%s ext=%s path=%s\n", id, identity, name, ext, fpath)
	}

	// 查 user_repository
	rows2, err := db.Query("SELECT id, identity, repository_identity, name, ext, parent_id FROM user_repository WHERE deleted_at = '0001-01-01 00:00:00' OR deleted_at IS NULL LIMIT 20")
	if err != nil {
		log.Fatal("query user_repository:", err)
	}
	defer rows2.Close()
	fmt.Println("\n--- user_repository ---")
	for rows2.Next() {
		var id, parentId int
		var identity, repoIdentity, name, ext string
		rows2.Scan(&id, &identity, &repoIdentity, &name, &ext, &parentId)
		fmt.Printf("  id=%d identity=%s repoIdentity=%s name=%s ext=%s parent_id=%d\n", id, identity, repoIdentity, name, ext, parentId)
	}

	// 查 share_basic
	rows3, err := db.Query("SELECT id, identity, repository_identity, user_repository_identity, expired_time, created_at FROM share_basic WHERE deleted_at = '0001-01-01 00:00:00' OR deleted_at IS NULL LIMIT 10")
	if err != nil {
		log.Fatal("query share_basic:", err)
	}
	defer rows3.Close()
	fmt.Println("\n--- share_basic ---")
	for rows3.Next() {
		var id, expiredTime int
		var identity, repoIdentity, userRepoIdentity, createdAt string
		rows3.Scan(&id, &identity, &repoIdentity, &userRepoIdentity, &expiredTime, &createdAt)
		fmt.Printf("  id=%d identity=%s repository_identity=%s user_repository_identity=%s expired_time=%d created_at=%s\n",
			id, identity, repoIdentity, userRepoIdentity, expiredTime, createdAt)
	}

	// 检查 MinIO
	fmt.Println("\n=== MinIO Check ===")
	minioClient, err := minio.New("192.168.204.130:9000", &minio.Options{
		Creds:  credentials.NewStaticV4("minio123", "minio123", ""),
		Secure: false,
	})
	if err != nil {
		log.Println("minio new:", err)
		return
	}

	// 检查 bucket 是否存在
	exists, err := minioClient.BucketExists(context.Background(), "wttest")
	if err != nil {
		fmt.Println("BucketExists error:", err)
	} else {
		fmt.Println("Bucket 'wttest' exists:", exists)
	}

	// 列出所有 buckets
	buckets, err := minioClient.ListBuckets(context.Background())
	if err != nil {
		fmt.Println("ListBuckets error:", err)
	} else {
		fmt.Println("All buckets:")
		for _, b := range buckets {
			fmt.Printf("  %s (created: %s)\n", b.Name, b.CreationDate)
		}
	}

	if exists {
		// 列出对象
		fmt.Println("\nObjects in 'wttest':")
		ctx := context.Background()
		objectCh := minioClient.ListObjects(ctx, "wttest", minio.ListObjectsOptions{
			Recursive: true,
		})
		count := 0
		for object := range objectCh {
			if object.Err != nil {
				fmt.Println("  List error:", object.Err)
				break
			}
			fmt.Printf("  %s (%d bytes)\n", object.Key, object.Size)
			count++
			if count >= 20 {
				fmt.Println("  ... (more objects)")
				break
			}
		}
		if count == 0 {
			fmt.Println("  (no objects found)")
		}
	}
}
