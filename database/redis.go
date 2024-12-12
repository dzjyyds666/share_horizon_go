package database

import (
	"ShareHorizon/config"
	"ShareHorizon/utils/log/logx"
	"fmt"
	"github.com/redis/go-redis/v9"
	"os"

	"context"
)

// 全局redis客户端
var RDB *redis.Client

const (
	Redis_Token_Key             = "token:%s"
	Redis_Captcha_Key           = "captcha:%s"
	Redis_Verification_Code_Key = "verification_code:%s"
	Redis_Register_Verify_Key   = "register_verify:%s"
)

// 初始化redis客户端
func InitRedis() {
	rdb := config.GlobalConfig.Redis
	RDB = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", rdb.Host, rdb.Port),
		Password: rdb.Password,
		DB:       rdb.DB,
	})

	_, err := RDB.Ping(context.Background()).Result()
	if err != nil {
		logx.GetLogger("SH").Errorf("Database|RedisConnect|FAIL|%v", err)
		os.Exit(1)
	} else {
		logx.GetLogger("SH").Info("Database|RedisConnect|SUCC")
	}
}
