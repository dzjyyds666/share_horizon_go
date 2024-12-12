package main

import (
	"ShareHorizon/config"
	_ "ShareHorizon/config"
	_ "ShareHorizon/database"
	"ShareHorizon/middlewares"
	"ShareHorizon/router"
	"ShareHorizon/utils/ascallArt"
	"ShareHorizon/utils/log/logx"
	"fmt"

	"github.com/gin-gonic/gin"
)

// main 程序入口

// @title 共享视界
// @version 0.1.0
// @description 共享视界后端，供app和web端调用

// @contact.name Aaron
// @contact.email duaaron519@gmail.com

// @host http://localhost:8888
// @BasePath /api/v1
func main() {
	c := gin.Default()

	// 全局异常处理，接收panic
	middlewares.Recovery(c)

	router.InitRouter(c)

	ascallArt.Showart()

	err := c.Run(fmt.Sprintf(":%d", config.GlobalConfig.ServerPort))
	if err != nil {
		logx.GetLogger("SH").Errorf("BootStrop|StartError|%v", err)
	}
}
