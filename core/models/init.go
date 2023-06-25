package models

import (
	"log"

	"core/internal/config"

	"github.com/go-redis/redis/v8"
	_ "github.com/go-sql-driver/mysql"
	"xorm.io/xorm"
)

func Init(c config.Config) *xorm.Engine {
	// engine, err := xorm.NewEngine("mysql", "root:1234@tcp(192.168.204.130:3306)/cloud-disk?charset=utf8mb4&parseTime=True&loc=Local")
	// engine, err := xorm.NewEngine(c.DataBase.Type, c.DataBase.Url)
	engine, err := xorm.NewEngine("mysql", c.Mysql.DataSource)
	if err != nil {
		log.Printf("Xorm New Engine Error:%v", err)
		return nil
	}
	return engine
}

func InitRedis(c config.Config) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     c.Redis.Addr,
		Password: c.Redis.Password,
		DB:       0,
		PoolSize: c.Redis.PoolSize,
	})
}
