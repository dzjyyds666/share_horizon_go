package handlers

import (
	"ShareHorizon/database"
	"ShareHorizon/models"
	"ShareHorizon/utils/log/logx"
	"ShareHorizon/utils/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"net/http"
	"strconv"
	"strings"
)

// GetMovieInfoPage
// @Summary 分页获取电影信息
// @Tags 电影
// @Accept json
// @Produce json
// @Param page query string false "页码"
// @Param limit query string false "每页数量"
// @Router /movie/list [get]
func GetMovieInfoPage(ctx *gin.Context) {
	page := ctx.Query("page")
	if len(page) <= 0 {
		page = "10"
	}
	pageInt, _ := strconv.Atoi(page)
	limit := ctx.Query("limit")
	if len(limit) <= 0 {
		limit = "1"
	}
	limitInt, _ := strconv.Atoi(limit)

	offset := (pageInt - 1) * limitInt
	var movies []models.MovieInfo

	result := database.MyDB.Preload("VideoType").Limit(limitInt).Offset(offset).Select("movie_id", "movie_name", "movie_photo", "movie_introduction", "movie_release_time", "movie_total_time", "movie_score").Find(&movies)
	if result.Error != nil {
		logx.GetLogger("SH").Errorf("Database|GetMovieInfoPage|Error|%v", result.Error)
		panic(result.Error)
	}

	ctx.JSON(http.StatusOK, response.NewResult(response.EnmuHttptatus.RequestFail, "获取成功", movies))
}

// GetMovieInfo
// @Summary 获取电影信息
// @Description 获取电影信息
// @Tags 电影
// @Accept json
// @Produce json
// @Param movie_id path string true "电影ID"
// @Router /movie/info/{movie_id} [get]
func GetMovieInfo(ctx *gin.Context) {
	movieId := ctx.Param("movie_id")
	var movie models.MovieInfo
	result := database.MyDB.Where("movie_id = ? and status = ?", movieId, models.MovieStatus.Normal).First(&movie)
	if result.Error != nil {
		logx.GetLogger("SH").Errorf("Database|GetMovieInfo|Error|%v", result.Error)
		panic(result.Error)
	}
	ctx.JSON(http.StatusOK, response.NewResult(response.EnmuHttptatus.RequestFail, "获取成功", movie))
}

type upload_movie_info struct {
	MovieName          string   `json:"movie_name"`
	MovieTotalTime     string   `json:"movie_total_time"`    // 总时长
	MovieReleaseTime   string   `json:"movie_release_time"`  // 上映时间
	MovieCountry       string   `json:"movie_country"`       // 国家
	MoviePhoto         string   `json:"movie_photo"`         // 封面
	MovieIntrodeuction string   `json:"movie_introdeuction"` // 简介
	MovieUrl           string   `json:"movie_url"`           // 播放地址
	TypeId             []string `json:"type_id"`             // 类型数组
}

// UploadMovieInfo
// @Summary 上传电影信息
// @Description 上传电影信息
// @Tags 电影
// @Accept json
// @Produce json
// @Params body upload_movie_info true "上传电影信息"
// @Router /movie/upload [post]
func UploadMovieInfo(ctx *gin.Context) {
	var movie upload_movie_info
	err := ctx.ShouldBindJSON(&movie)
	if err != nil {
		logx.GetLogger("SH").Errorf("Handler|UploadMovieInfo|ParamsError|%v", err)
		ctx.JSON(http.StatusOK, response.NewResult(response.EnmuHttptatus.ParamError, "参数错误", nil))
		ctx.Abort()
	}

	var movieInfo models.MovieInfo
	// 生成movie_id
	newUUID, _ := uuid.NewUUID()
	movieId := strings.ReplaceAll(newUUID.String(), "-", "")
	movieInfo.MovieId = movieId
	movieInfo.MovieName = movie.MovieName
	movieInfo.MovieTotalTime = movie.MovieTotalTime
	movieInfo.MovieReleaseTime = movie.MovieReleaseTime
	movieInfo.MovieCountry = movie.MovieCountry
	movieInfo.MoviePhoto = movie.MoviePhoto
	movieInfo.MovieIntrodeuction = movie.MovieIntrodeuction
	movieInfo.MovieUrl = movie.MovieUrl
	movieInfo.Status = models.MovieStatus.Normal

	err = database.MyDB.Create(&movie).Error
	if err != nil {
		if err == gorm.ErrDuplicatedKey { // 键重复
			logx.GetLogger("SH").Errorf("Handler|UploadMovieInfo|DuplicateKey|%v", err)
			ctx.JSON(http.StatusOK, response.NewResult(response.EnmuHttptatus.RequestFail, "电影id重复，请重新上传", nil))
			ctx.Abort()
		} else {
			logx.GetLogger("SH").Errorf("Handler|UploadMovieInfo|CreateMovieInfoError|%v", err)
			panic(err)
		}
	}

	// 插入关联的 VideoType 数据到中间表
	var videoTypes []models.VideoType
	for _, typeId := range movie.TypeId {
		var videoType models.VideoType
		result := database.MyDB.First(&videoType, "id = ?", typeId)
		if result.Error != nil {
			logx.GetLogger("SH").Errorf("Handler|UploadMovieInfo|VideoTypeError|%v", result.Error)
			ctx.JSON(http.StatusOK, response.NewResult(response.EnmuHttptatus.RequestFail, "视频类型不存在", nil))
			ctx.Abort()
			return
		}
		videoTypes = append(videoTypes, videoType)
	}

	// 关联 VideoType 到 MovieInfo
	err = database.MyDB.Model(&movieInfo).Association("VideoType").Append(videoTypes)
	if err != nil {
		logx.GetLogger("SH").Errorf("Handler|UploadMovieInfo|AppendVideoTypeError|%v", err)
		panic(err)
	}

	ctx.JSON(http.StatusOK, response.NewResult(response.EnmuHttptatus.RequestSuccess, "上传成功", movieInfo))
}
