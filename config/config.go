package config

import (
	"ShareHorizon/utils/log/logx"
	"github.com/spf13/viper"
	"os"
)

// Config 结构体表示应用程序的配置
type Config struct {
	AppName    string `mapstructure:"app_name"`
	AppVersion string `mapstructure:"app_version"`
	ServerPort int    `mapstructure:"server_port"`
	MySQL      MySQLConfig
	JWT        JwtConfig
	Redis      RedisConfig
	Email      EmailConfig
	//AI         AIConfig
	//Cos        CosConfig
}

// MySQLConfig 结构体表示 MySQL 的配置
type MySQLConfig struct {
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	DBName   string `mapstructure:"db_name"`
}

// JwtConfig 结构体表示 JWT 的配置
type JwtConfig struct {
	SecretKey      string `mapstructure:"secret_key"`
	ExpirationTime int64  `mapstructure:"expiration_time"`
}

// redis 配置
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       []int  `mapstructure:"db"`
}

type EmailConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	From     string `mapstructure:"from"`
	Alias    string `mapstructure:"alias"`
}

type AIConfig struct {
	ApiKey    string `mapstructure:"api_key"`
	SecretKey string `mapstructure:"secret_key"`
	Endpoint  string `mapstructure:"endpoint"`
}

type CosConfig struct {
	AliasName string   `mapstructure:"alias_name"`
	AccessId  string   `mapstructure:"accessId"`
	SecretKey string   `mapstructure:"secretKey"`
	IamRoleid string   `mapstructure:"iamRoleid"`
	Region    string   `mapstructure:"region"`
	Buckets   []string `mapstructure:"buckets"`
	Url       string   `mapstructure:"url"`
}

var GlobalConfig Config

func init() {
	// 读取配置文件
	err := LoggingConfig()
	if err != nil {
		logx.GetLogger("SH").Errorf("Config|InitConfig|FAIL|%v", err)
		os.Exit(1)
	}
	logx.GetLogger("SH").Info("Config|InitConfig|SUCC")
}

func LoggingConfig() error {

	configFilePath := "./config/config.toml"

	// 使用viper读取配置文件
	viper.SetConfigType("toml")
	viper.SetConfigFile(configFilePath)

	err := viper.ReadInConfig()
	if err != nil {
		logx.GetLogger("SH").Errorf("Config|LoggingConfig|ReadError|%v", err)
		return err
	}

	// 将配置文件绑定结构体
	err = viper.Unmarshal(&GlobalConfig)
	if err != nil {
		logx.GetLogger("SH").Errorf("Config|LoggingConfig|JsonUnmarshalError|%v", err)
		return err
	}
	return nil
}
