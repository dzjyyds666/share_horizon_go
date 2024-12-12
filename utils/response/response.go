package response

import "time"

type Result struct {
	Code HttpStatus  `json:"code,omitempty"`
	Data interface{} `json:"data,omitempty"`
	Msg  string      `json:"msg,omitempty"`
	Time string      `json:"time,omitempty"`
}

func NewResult(code HttpStatus, msg string, data interface{}) *Result {
	return &Result{
		Code: code,
		Data: data,
		Msg:  msg,
		Time: time.Now().Format("2006-01-02 15:04:05"),
	}
}

type HttpStatus int

var EnmuHttptatus = struct {
	RequestSuccess HttpStatus
	RequestFail    HttpStatus

	//User Error
	UserNotExist     HttpStatus
	TokenInvalid     HttpStatus // Token 无效
	TokenExpired     HttpStatus // Token 过期
	UserIsRegistered HttpStatus // 用户已注册

	RedisError      HttpStatus
	MySQLWriteError HttpStatus // 写入数据库失败

	SystemError HttpStatus // 系统异常

	ParamError    HttpStatus // 参数错误
	PasswordError HttpStatus // 密码错误

	ArticleTypeExists HttpStatus // 文章类型已存在
}{
	RequestSuccess: 10200,
	RequestFail:    10201,

	UserNotExist:     10301,
	TokenInvalid:     10302,
	TokenExpired:     10303,
	UserIsRegistered: 10304,

	RedisError:      10401,
	MySQLWriteError: 10402,

	SystemError: 10500,

	ParamError:    10601,
	PasswordError: 10602,

	ArticleTypeExists: 10701,
}
