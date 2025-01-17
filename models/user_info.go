package models

import (
	"time"
)

type UserInfo struct {
	UserID             string    `gorm:"column:user_id;primaryKey;size:12;not null"`     // 用户ID
	NickName           string    `gorm:"column:nick_name;size:20;not null;unique"`       // 用户名
	UserPassword       string    `gorm:"column:user_password;size:15;not null"`          // 密码
	Email              string    `gorm:"column:email;size:30;unique;not null;unique"`    // 邮箱
	Sex                int8      `gorm:"column:sex;not null"`                            // 性别 0:女 1:男 2:机器人
	Birthday           string    `gorm:"column:birthday;size:10"`                        // 生日
	PersonIntroduction string    `gorm:"column:person_introduction;size:200"`            // 个人介绍
	Avatar             string    `gorm:"column:avatar;size:200"`                         // 头像
	CreateTime         time.Time `gorm:"column:create_time;autoCreateTime;not null"`     // 创建时间
	LastLoginTime      time.Time `gorm:"column:last_login_time;autoUpdateTime;not null"` // 最后登录时间
	LastLoginIP        string    `gorm:"column:last_login_ip;size:20;not null"`          // 最后登录IP
	Status             int8      `gorm:"column:status;default:1;not null"`               // 状态 0:用户异常 1:正常
	IsDeleted          int8      `gorm:"column:is_deleted;default:1;not null"`           // 逻辑删除 0:删除 1:未删除
	Experience         int       `gorm:"column:experience;default:0;not null"`           // 经验值
	Threme             int8      `gorm:"column:threme;default:1;not null"`               // 主题 1:亮色 2:暗色
	Role               string    `gorm:"column:role;size:10;not null"`                   // 角色
}

// TableName 设置表名
func (UserInfo) TableName() string {
	return "user_info"
}

var Role = struct {
	User  string
	Admin string
}{
	User:  "user",
	Admin: "admin",
}
