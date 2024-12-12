package jwt

import (
	"ShareHorizon/config"
	"ShareHorizon/utils/log/logx"
	"strings"
	"time"

	"github.com/golang-jwt/jwt"
)

type JWT struct {
	JwtSecretKey   []byte // jwt加密的密钥
	ExpirationTime int64  // 有效期
}

type Claims struct {
	UserId string `json:"user_id"`
	jwt.StandardClaims
}

func NewJWTUtils() *JWT {
	return &JWT{
		JwtSecretKey:   []byte(config.GlobalConfig.JWT.SecretKey),
		ExpirationTime: config.GlobalConfig.JWT.ExpirationTime,
	}
}

func (j *JWT) CreateJWT(UserId string) string {

	//创建声明
	claims := &Claims{
		UserId: UserId,
		StandardClaims: jwt.StandardClaims{
			// 过期时间 ,使用time.Now().Add() 添加时间
			ExpiresAt: time.Now().Add(time.Hour * time.Duration(j.ExpirationTime)).Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	//生成token字符串
	tokenStr, err := token.SignedString(j.JwtSecretKey)
	if err != nil {
		panic("生成token失败:" + err.Error())
	}
	return "Bearer " + tokenStr
}

func (j *JWT) ParseJWT(tokenStr string) (*Claims, error) {

	tokenStr = strings.Replace(tokenStr, "Bearer ", "", 1)

	logx.GetLogger("logx").Infof("ParseJWT|tokenStr: %v", tokenStr)

	//解析token
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return j.JwtSecretKey, nil
	})
	if err != nil {
		return nil, err
	}
	return token.Claims.(*Claims), nil
}
