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

			movie := public.Group("/movie")
			{
				movie.GET("/list", handlers.GetMovieInfoPage)
				movie.GET("/info/:movie_id", handlers.GetMovieInfo)
				movie.POST("/upload", handlers.UploadMovieInfo)
			}
		}

		auth := v1.Group("")
		auth.Use(middlewares.TokenVerify())
		{
			auth.GET("/logout", handlers.Logout)

			oss := auth.Group("/oss")
			{
				oss.GET("/applayUpload", handlers.ApplayUpload)
				oss.POST("/upload/applayUpload", handlers.ApplayUpload)
				oss.POST("/upload/putFile/:fid", handlers.PutFile)
				oss.POST("/upload/initMultipartUpload", handlers.InitMultipartFile)
				oss.POST("/upload/multipart/:fid", handlers.MultipartUploadFile)
				oss.POST("/upload/multipart/complete/:fid", handlers.CompleteMultipartUpload)
				oss.POST("/upload/multipart/abort/:fid", handlers.AbortMultipartUpload)
			}
		}
	}

}
