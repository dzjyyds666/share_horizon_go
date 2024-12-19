package router

import (
	"ShareHorizon/handlers"
	"ShareHorizon/middlewares"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func InitRouter(c *gin.Engine) {

	c.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	v1 := c.Group("/api/v1")
	{
		public := v1.Group("")
		{
			public.GET("/loginPass", handlers.LoginByPass)
			public.POST("/loginVer", handlers.LoginByVerfiyCode)
			public.POST("/register", handlers.Register)
			public.GET("/sendEmail", handlers.SendVerifyCode)
			public.GET("/getCaptcha", handlers.GetCaptchaCode)
		}

		//test := v1.Group("/test")
		//{
		//	//test.GET("/getFileInfo", handlers.GetFileInfoTest)
		//}

		auth := v1.Group("")
		auth.Use(middlewares.TokenVerify())
		{
			auth.GET("/logout", handlers.Logout)
			auth.GET("/applayUpload", handlers.ApplayUpload)
		}
	}

}
