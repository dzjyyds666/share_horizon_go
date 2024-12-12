package router

import (
	"ShareHorizon/handlers"
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
			public.GET("/sendEmail", handlers.SendEmail)
			public.GET("/getCaptcha", handlers.GetCaptcha)
		}

		auth := v1.Group("")
		{
			auth.GET("/logout", handlers.Logout)
		}
	}

}
