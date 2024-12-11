package middlewares

import (
	"ShareHorizon/utils/log/logx"
	"ShareHorizon/utils/response"
	"github.com/gin-gonic/gin"
	"net/http"
)

// Recovery 全局异常处理
func Recovery(r *gin.Engine) {
	r.Use(gin.CustomRecovery(func(c *gin.Context, recovered any) {
		logx.GetLogger("SH").Errorf("SystemError|%v", recovered)
		c.JSON(http.StatusOK, response.NewResult(response.EnmuHttptatus.SystemError, "系统异常,请稍后再试", recovered))
	}))
}
