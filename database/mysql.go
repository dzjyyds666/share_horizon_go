package database

import (
	"ShareHorizon/config"
	"ShareHorizon/models"
	"ShareHorizon/utils/log/logx"
	"fmt"
	"gorm.io/gorm/logger"
	"os"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// 全局mysql客户端
var MyDB *gorm.DB

func InitMySQL() {
	sql := config.GlobalConfig.MySQL
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		sql.Username,
		sql.Password,
		sql.Host,
		sql.Port,
		sql.DBName)
	var err error
	MyDB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		// 开启日志
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		logx.GetLogger("SH").Errorf("Database|MySqlConnect|FAIL|%v", err)
		os.Exit(1)
	}

	logx.GetLogger("SH").Info("Database|MySqlConnect|SUCC")

	err = MyDB.AutoMigrate(&models.UserInfo{}, &models.MovieInfo{}, &models.VideoType{}) // 创建表
	if err != nil {
		logx.GetLogger("SH").Errorf("Database|MySqlAuToMigraye|Error|%v", err)
		os.Exit(1)
	}
}
