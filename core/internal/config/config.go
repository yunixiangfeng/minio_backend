package config

import "github.com/zeromicro/go-zero/rest"

type Config struct {
	rest.RestConf

	// DataBase struct {
	// 	Type         string
	// 	Url          string
	// 	MaxOpenConns int
	// 	MaxIdleConns int
	// 	ShowSql      bool
	// }
	Mysql struct {
		DataSource string
	}
	// Redis struct {
	// 	Addr string
	// }
	Redis struct {
		Addr     string
		PoolSize int
		Password string
	}
}
