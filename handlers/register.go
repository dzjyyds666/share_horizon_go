package handlers

import (
	"ShareHorizon/database"
	"ShareHorizon/models"
	"ShareHorizon/utils/email"
	"ShareHorizon/utils/log/logx"
	"ShareHorizon/utils/response"
	"fmt"
	"github.com/dlclark/regexp2"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mojocn/base64Captcha"
	"golang.org/x/crypto/bcrypt"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

type registerInfo struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	VerifyCode  string `json:"verify_code"`
	CaptchaId   string `json:"captcha_id"`
	CaptchaCode string `json:"captcha_code"`
}

var passwordRegex = regexp2.MustCompile("^(a-zA-Z0-9.@_){8,12}$", regexp2.None)

func Register(ctx *gin.Context) {
	var registerinfo registerInfo

	err := ctx.ShouldBindJSON(&registerinfo)
	if err != nil {
		logx.GetLogger("SH").Errorf("Handler|Register|ParamsError|%v", err)
		ctx.JSON(http.StatusOK, response.NewResult(response.EnmuHttptatus.ParamError, "参数错误", nil))
		return
	}

	matchString, err := passwordRegex.MatchString(registerinfo.Password)
	if err != nil {
		logx.GetLogger("SH").Errorf("Handler|Register|PasswordRegexError|%v", err)
		panic(err)
	}
	if matchString == false {
		logx.GetLogger("SH").Errorf("Handler|Register|PasswordRegexError|%v", err)
		ctx.JSON(http.StatusOK, response.NewResult(response.EnmuHttptatus.ParamError, "密码包含字母数字，特殊字符（.、_、@）", nil))
		return
	}

	get := database.RDB[0].Get(ctx, fmt.Sprintf(database.Redis_Captcha_Key, registerinfo.CaptchaId))
	if get.Err() != nil {
		logx.GetLogger("SH").Errorf("Handler|Register|GetCaptchaError|%v", err)
		panic(get.Err())
	}
	if get.Val() != registerinfo.CaptchaCode {
		logx.GetLogger("SH").Errorf("Handler|Register|CaptchaNotMatch")
		ctx.JSON(http.StatusOK, response.NewResult(response.EnmuHttptatus.ParamError, "图片验证码错误", nil))
		return
	}

	get = database.RDB[0].Get(ctx, fmt.Sprintf(database.Redis_Register_Verify_Key, registerinfo.Email))
	if get.Err() != nil {
		logx.GetLogger("SH").Errorf("Handler|Register|GetVerifyCodeError|%v", err)
		panic(get.Err())
	}
	if get.Val() != registerinfo.VerifyCode {
		logx.GetLogger("SH").Errorf("Handler|Register|VerifyCodeNotMatch")
		ctx.JSON(http.StatusOK, response.NewResult(response.EnmuHttptatus.ParamError, "邮箱验证码错误", nil))
		return
	}

	var userInfo models.UserInfo
	newUUID, err := uuid.NewUUID()
	if err != nil {
		logx.GetLogger("SH").Errorf("Handler|Register|NewUUIDError|%v", err)
		panic(err)
	}

	userInfo.UserID = strings.ReplaceAll(newUUID.String(), "-", "")
	userInfo.Email = registerinfo.Email
	password, _ := bcrypt.GenerateFromPassword([]byte(registerinfo.Password), bcrypt.DefaultCost)
	userInfo.UserPassword = string(password)
	userInfo.Sex = models.Sex.Robot
	userInfo.NickName = "用户" + userInfo.Email[:6]
	userInfo.CreateTime = time.Now()
	userInfo.Role = models.Role.User

	err = database.MyDB.Create(&userInfo).Error
	if err != nil {
		logx.GetLogger("SH").Errorf("Handler|Register|CreateUserError|%v", err)
		ctx.JSON(http.StatusOK, response.NewResult(response.EnmuHttptatus.RequestFail, "注册失败", err))
	}

	ctx.JSON(http.StatusOK, response.NewResult(response.EnmuHttptatus.RequestSuccess, "注册成功", models.UserInfo{
		Email:        userInfo.Email,
		UserPassword: userInfo.UserPassword,
	}))

}

// 获取图形验证码
func GetCaptchaCode(ctx *gin.Context) {
	driver := base64Captcha.NewDriverDigit(80, 200, 5, 0.8, 75)
	store := base64Captcha.DefaultMemStore

	//生成图形验证码
	captcha := base64Captcha.NewCaptcha(driver, store)

	id, b64s, answer, err := captcha.Generate()
	if err != nil {
		panic("获取图片验证码失败" + err.Error())
	}

	// 使用redis存取验证码
	err = database.RDB[0].Set(ctx, fmt.Sprintf(database.Redis_Captcha_Key, id), answer, time.Minute*5).Err()

	if err != nil {
		panic("redis存取验证码失败" + err.Error())
	}

	ctx.JSON(http.StatusOK, response.NewResult(response.EnmuHttptatus.RequestSuccess, "获取验证码成功", map[string]string{
		"id":      id,
		"captcha": b64s,
	}))
}

// 发送注册验证码
func SendVerifyCode(ctx *gin.Context) {

	to := ctx.Query("email")

	// 生成随机验证码
	verificationCode := GenerateVerificationCode(6)

	// 把验证码存入redis
	ok, err := database.RDB[0].SetNX(ctx, fmt.Sprintf(database.Redis_Verification_Code_Key, to), verificationCode, time.Minute*5).Result()
	if err != nil {
		panic("redis存取验证码失败" + err.Error())
	}

	if !ok {
		ctx.JSON(http.StatusOK, response.NewResult(response.EnmuHttptatus.RequestSuccess, "验证码已发送，请稍后再试", nil))
		return
	}

	subject := "验证码"
	body := fmt.Sprintf(
		`<p>您的验证码是: %s</p>
		<p>请在5分钟内完成注册</p>
		<p>请不要回复此邮件</p>`, verificationCode)

	err = email.SendEmail(to, subject, body)
	if err != nil {
		panic("发送邮件失败:" + err.Error())
	}

	ctx.JSON(http.StatusOK, response.NewResult(response.EnmuHttptatus.RequestSuccess, "验证码已发送，请查收", nil))
}

func GenerateVerificationCode(length int) string {
	const charset = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

	rand.Seed(time.Now().UnixNano())

	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}
