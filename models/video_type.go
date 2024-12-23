package models

type VideoType struct {
	VideoTypeId   string `gorm:"column:video_type_id;primaryKey;size:36;not null"`
	VideoTypeName string `gorm:"column:video_type_name;size:50;not null"`
	Status        string `gorm:"column:status;size:50;not null"`

	movie_info []MovieInfo `gorm:"many2many:movie_types;"`
}

func (VideoType) TableName() string {
	return "video_type"
}
