package main

import (
	"ShareHorizon/config"
	_ "ShareHorizon/config"
	_ "ShareHorizon/database"
	"ShareHorizon/middlewares"
	"ShareHorizon/utils/ascallArt"
	"ShareHorizon/utils/log/logx"
	"fmt"

	"github.com/gin-gonic/gin"
)

// main 程序入口
func main() {
	c := gin.Default()

	// 全局异常处理，接收panic
	middlewares.Recovery(c)

	ascallArt.Showart()

	err := c.Run(fmt.Sprintf(":%d", config.GlobalConfig.ServerPort))
	if err != nil {
		logx.GetLogger("SH").Errorf("BootStrop|StartError|%v", err)
	}
}
