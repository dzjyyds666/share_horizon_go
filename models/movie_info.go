package models

type MovieInfo struct {
	MovieId            string `gorm:"column:movie_id;primaryKey;size:36;not null"`
	MovieName          string `gorm:"column:movie_name;size:50;not null"`
	MovieTotalTime     string `gorm:"column:movie_total_time;size:50;not null"`      // 总时长
	MovieReleaseTime   string `gorm:"column:movie_release_time;size:50;not null"`    // 上映时间
	MovieScore         string `gorm:"column:movie_score;size:50;not null;default:0"` // 评分
	MovieCountry       string `gorm:"column:movie_country;size:50;not null"`         // 国家
	MoviePhoto         string `gorm:"column:movie_photo;size:200;not null"`          // 封面
	MovieIntrodeuction string `gorm:"column:movie_introduction;size:200;not null"`   // 简介
	MovieUrl           string `gorm:"column:movie_url;size:200;not null"`            // 播放地址
	Status             string `gorm:"column:status;size:1;not null;default:1"`       // 状态

	VideoType []VideoType `gorm:"many2many:movie_types;"`
}

func (MovieInfo) TableName() string {
	return "movie_info"
}

var MovieStatus = struct {
	Normal string
	Delete string
	Hidden string
}{
	Normal: "1",
	Delete: "2",
	Hidden: "3",
}
