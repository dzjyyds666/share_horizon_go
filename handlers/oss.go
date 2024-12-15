package handlers

import (
	"ShareHorizon/database"
	"ShareHorizon/models"
	"ShareHorizon/utils/aws"
	"ShareHorizon/utils/log/logx"
	"ShareHorizon/utils/response"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"net/http"
	"os"
	"time"
)

func GetFileInfoTest(ctx *gin.Context) {
	path := "/Users/zhijundu/Downloads/cosv20241113.zip"
	local, err := aws.GetUploadInfoFromLocal(path)
	if err != nil {
		ctx.JSON(400, gin.H{
			"code": 400,
			"msg":  err.Error(),
		})
		return
	}
	local.WithFid("test-001")
	client, err := aws.GetS3Client("minio-oss")
	if err != nil {
		ctx.JSON(400, gin.H{
			"code": 400,
			"msg":  err.Error(),
		})
	}
	file, _ := os.Open(path)
	ok, err := aws.PutFile(local, file, "oss-file-dev", client)
	if !ok {
		ctx.JSON(400, gin.H{
			"code": 400,
			"msg":  err.Error(),
		})
		return
	}
	ctx.JSON(200, gin.H{
		"code": 200,
		"msg":  "success",
		"data": local,
	})
}

type ApplayUploadInfo struct {
	StorageId   string `json:"storage_id"`
	PartitionId string `json:"partition_id"`
	DirectoryId string `json:"directory_id"`
	UserId      string `json:"user_id"`
	Bucket      string `json:"bucket"`
}

func ApplayUpload(ctx *gin.Context) {
	userId, _ := ctx.Get("user_id")
	var userInfo *models.UserInfo
	res := database.MyDB.Where("user_id = ?", userId).First(&userInfo)
	if res.Error != nil {
		logx.GetLogger("SH").Errorf("Handler|ApplayUpload|GetUserInfoError|%v", res.Error)
		ctx.JSON(http.StatusBadRequest, response.NewResult(response.EnmuHttptatus.RequestFail, "获取用户信息失败", nil))
		ctx.Abort()
	}

	if userInfo.Role == models.Role.Guest {
		ctx.JSON(http.StatusBadRequest, response.NewResult(response.EnmuHttptatus.RequestFail, "你没有上传文件的权利", nil))
		ctx.Abort()
	}

	storageId := ctx.GetHeader("OssX-storageId")
	if storageId == "" {
		storageId = "minio-oss"
	}

	bucket := ctx.GetHeader("OssX-bucket")
	if bucket == "" {
		bucket = "oss-file-dev"
	}

	// 获取partitionId
	partitionId := ctx.GetHeader("OssX-partitionId")
	if partitionId == "" {
		partitionId = "default"
	}

	// 获取文件夹id
	directoryId := ctx.GetHeader("OssX-directoryId")
	if directoryId == "" {
		ctx.JSON(http.StatusBadRequest, response.NewResult(response.EnmuHttptatus.RequestFail, "获取文件夹id失败", nil))
		ctx.Abort()
	}

	tmp := fmt.Sprintf("%s-%s-%s-%s", userId, partitionId, directoryId, time.Now().Format("20060102150405"))
	hash := md5.New()
	hash.Write([]byte(tmp))
	fid := hex.EncodeToString(hash.Sum(nil))
	fid = "v1-" + fid

	applayUploadInfo := &ApplayUploadInfo{
		PartitionId: partitionId,
		DirectoryId: directoryId,
		UserId:      userInfo.UserID,
	}

	marshal, err := json.Marshal(applayUploadInfo)
	if err != nil {
		logx.GetLogger("SH").Errorf("Handler|ApplayUpload|MarshalError|%v", err)
		panic(err)
	}

	// 把fid存入redis
	err = database.RDB.Set(ctx, fmt.Sprintf(database.Redis_Applay_Upload_Fid, fid), marshal, 0).Err()
	if err != nil {
		logx.GetLogger("SH").Errorf("Handler|ApplayUpload|SetRedisError|%v", err)
		panic(err)
	}
	ctx.JSON(http.StatusOK, response.NewResult(response.EnmuHttptatus.RequestSuccess, "申请上传成功", fid))
}

func PutFile(ctx *gin.Context) {
	file, err := ctx.FormFile("file")
	if err != nil {
		logx.GetLogger("SH").Errorf("Handler|PutFile|GetFileError|%v", err)
		ctx.JSON(http.StatusBadRequest, response.NewResult(response.EnmuHttptatus.RequestFail, "获取文件失败", nil))
		ctx.Abort()
	}
	open, err := file.Open()
	if err != nil {
		logx.GetLogger("SH").Errorf("Handler|PutFile|OpenFileError|%v", err)
		ctx.JSON(http.StatusBadRequest, response.NewResult(response.EnmuHttptatus.RequestFail, "打开文件失败", nil))
	}
	defer open.Close()

	fid := ctx.PostForm("fid")
	if fid == "" {
		logx.GetLogger("SH").Errorf("Handler|PutFile|Fid can not be empty")
		ctx.JSON(http.StatusBadRequest, response.NewResult(response.EnmuHttptatus.RequestFail, "fid不能为空", nil))
		ctx.Abort()
	}

	fileName := ctx.PostForm("file_name")
	if fileName == "" {
		logx.GetLogger("SH").Errorf("Handler|PutFile|FileName can not be empty")
		ctx.JSON(http.StatusBadRequest, response.NewResult(response.EnmuHttptatus.RequestFail, "文件名不能为空", nil))
		ctx.Abort()
	}

	info, err := aws.GetUploadInfoFromStream(open, fileName)
	if err != nil {
		logx.GetLogger("SH").Errorf("Handler|PutFile|GetUploadInfoFromStreamError|%v", err)
		ctx.JSON(http.StatusBadRequest, response.NewResult(response.EnmuHttptatus.RequestFail, "获取文件信息失败", nil))
		ctx.Abort()
	}

	// 从redis中获取fid对应的上传信息
	value, err := database.RDB.Get(ctx, fmt.Sprintf(database.Redis_Applay_Upload_Fid, fid)).Result()
	if err != nil {
		if err == redis.Nil {
			logx.GetLogger("SH").Errorf("Handler|PutFile|FidInfoNotExistError|%v", err)
			ctx.JSON(http.StatusBadRequest, response.NewResult(response.EnmuHttptatus.RequestFail, "fid不存在，请重新申请", nil))
			ctx.Abort()
		} else {
			logx.GetLogger("SH").Errorf("Handler|PutFile|GetFidInfoFromRedisError|%v", err)
			ctx.JSON(http.StatusBadRequest, response.NewResult(response.EnmuHttptatus.RequestFail, "获取fid对应的上传信息失败", nil))
			ctx.Abort()
		}
	}

	var applayUploadInfo ApplayUploadInfo
	json.Unmarshal([]byte(value), &applayUploadInfo)
	info.WithFid(fid).WithKey(applayUploadInfo.PartitionId, applayUploadInfo.DirectoryId, applayUploadInfo.UserId)

	putFile, err := aws.PutFile(info, open, applayUploadInfo.StorageId, applayUploadInfo.Bucket)
	if !putFile {
		logx.GetLogger("SH").Errorf("Handler|PutFile|PutFileError|%v", err)
		ctx.JSON(http.StatusBadRequest, response.NewResult(response.EnmuHttptatus.RequestFail, "上传文件失败", nil))
		ctx.Abort()
	}
	logx.GetLogger("SH").Infof("Handler|PutFile|PutFileSuccess|%v", info)
	ctx.JSON(http.StatusOK, response.NewResult(response.EnmuHttptatus.RequestSuccess, "上传文件成功", nil))
}
