package handlers

import (
	"ShareHorizon/database"
	"ShareHorizon/models"
	"ShareHorizon/utils/aws"
	"ShareHorizon/utils/log/logx"
	"ShareHorizon/utils/response"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	s3aws "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"io"
	"net/http"
	"strconv"
	"time"
)

type ApplayUploadInfo struct {
	StorageId   string `json:"storage_id"`
	PartitionId string `json:"partition_id"`
	DirectoryId string `json:"directory_id"`
	UserId      string `json:"user_id"`
	Bucket      string `json:"bucket"`
}

// 生成32字节的随机key
func generateKey256() string {
	key := make([]byte, 32)
	io.ReadFull(rand.Reader, key)
	return base64.StdEncoding.EncodeToString(key)
}

func Excrypt(data []byte, key []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func Decrypt(ciphertextBase64, key []byte) ([]byte, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(string(ciphertextBase64))
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
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

	storageId := ctx.GetHeader(aws.OssHeaders.StorageId.ToString())
	if len(storageId) <= 0 {
		storageId = "minio-oss"
	}

	bucket := ctx.GetHeader(aws.OssHeaders.BucketId.ToString())
	if len(bucket) <= 0 {
		bucket = "oss-file-dev"
	}

	// 获取文件夹id
	directoryId := ctx.GetHeader(aws.OssHeaders.DirectoryId.ToString())
	if len(directoryId) <= 0 {
		ctx.JSON(http.StatusBadRequest, response.NewResult(response.EnmuHttptatus.RequestFail, "directoryId不可以为空", nil))
		ctx.Abort()
	}

	tmp := fmt.Sprintf("%s-%s-%s", userId, directoryId, time.Now().Format("20060102150405"))

	key := generateKey256()
	fid, err := Excrypt([]byte(tmp), []byte(key))
	if err != nil {
		logx.GetLogger("SH").Errorf("Handler|ApplayUpload|ExcryptError|%v", err)
		ctx.JSON(http.StatusBadRequest, response.NewResult(response.EnmuHttptatus.RequestFail, "上传文件签名失败", nil))
	}

	applayUploadInfo := &ApplayUploadInfo{
		StorageId:   storageId,
		DirectoryId: directoryId,
		UserId:      userInfo.UserID,
		Bucket:      bucket,
	}

	marshal, err := json.Marshal(applayUploadInfo)
	if err != nil {
		logx.GetLogger("SH").Errorf("Handler|ApplayUpload|MarshalError|%v", err)
		panic(err)
	}

	// 把fid存入redis,设置5分钟后过期
	err = database.RDB.Set(ctx, fmt.Sprintf(database.Redis_Applay_Upload_Fid, fid), marshal, time.Minute*5).Err()
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
			ctx.JSON(http.StatusBadRequest, response.NewResult(response.EnmuHttptatus.RequestFail, "Fid过期，请重新申请", nil))
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

// 初始化分片上传
func InitMultipartFile(ctx *gin.Context) {

	var uploadInfo aws.UploadInfo
	err := ctx.ShouldBindJSON(&uploadInfo)
	if err != nil {
		logx.GetLogger("SH").Errorf("Handler|InitMultipartFile|ParamsError|%v", err)
		ctx.JSON(http.StatusBadRequest, response.NewResult(response.EnmuHttptatus.ParamError, "参数错误", nil))
		ctx.Abort()
	}

	// 查询redis是否存在fid信息
	result, err := database.RDB.Get(ctx, fmt.Sprintf(database.Redis_Applay_Upload_Fid, uploadInfo.Fid)).Result()
	if err != nil {
		if err == redis.Nil {
			logx.GetLogger("SH").Errorf("Handler|InitMultipartFile|FidInfoNotExistError|%v", err)
			ctx.JSON(http.StatusBadRequest, response.NewResult(response.EnmuHttptatus.RequestFail, "Fid过期，请重新申请", nil))
			ctx.Abort()
		} else {
			logx.GetLogger("SH").Errorf("Handler|InitMultipartFile|GetFidInfoFromRedisError|%v", err)
			panic(err)
		}
	}

	var applayUpload ApplayUploadInfo
	err = json.Unmarshal([]byte(result), &applayUpload)
	if err != nil {
		logx.GetLogger("SH").Errorf("Handler|InitMultipartFile|UnmarshalError|%v", err)
		panic(err)
	}

	//获取s3客户端
	s3Client, err := aws.GetS3Client(applayUpload.StorageId)
	if err != nil {
		logx.GetLogger("SH").Errorf("Handler|InitMultipartFile|GetS3ClientError|%v", err)
		panic(err)
	}

	uploadInfo.WithKey(applayUpload.PartitionId, applayUpload.DirectoryId, applayUpload.UserId)

	upload, err := aws.InitMultUpload(applayUpload.Bucket, s3Client.S3Client, uploadInfo)
	if err != nil {
		logx.GetLogger("SH").Errorf("Handler|InitMultipartFile|InitMultUploadError|%v", err)
		ctx.JSON(http.StatusBadRequest, response.NewResult(response.EnmuHttptatus.RequestFail, "初始化分片上传失败", err))
	}

	ctx.JSON(http.StatusOK, response.NewResult(response.EnmuHttptatus.RequestSuccess, "初始化分片上传成功", map[string]string{
		"upload_id": s3aws.ToString(upload.UploadId),
	}))
}

func MultipartUploadFile(ctx *gin.Context) {
	var uploadInfo aws.UploadInfo
	err := ctx.ShouldBindJSON(&uploadInfo)
	if err != nil {
		logx.GetLogger("SH").Errorf("Handler|MultipartUploadFile|ParamsError|%v", err)
		ctx.JSON(http.StatusBadRequest, response.NewResult(response.EnmuHttptatus.ParamError, "参数错误", nil))
		ctx.Abort()
	}

	// 查询是否fid是否存在
	err = database.RDB.Get(ctx, fmt.Sprintf(database.Redis_Applay_Upload_Fid, uploadInfo.Fid)).Err()
	if errors.Is(err, redis.Nil) {
		logx.GetLogger("SH").Errorf("Handler|MultipartUploadFile|FidInfoNotExistError|%v", err)
		ctx.JSON(http.StatusBadRequest, response.NewResult(response.EnmuHttptatus.RequestFail, "Fid过期，请重新申请", nil))
		ctx.Abort()
	}

	if err != nil {
		logx.GetLogger("SH").Errorf("Handler|MultipartUploadFile|GetFidInfoFromRedisError|%v", err)
		panic(err)
	}

	partBytes, err := ctx.FormFile("part_bytes")
	if err != nil {
		logx.GetLogger("SH").Errorf("Handler|MultipartUploadFile|GetPartBytesError|%v", err)
		ctx.JSON(http.StatusBadRequest, response.NewResult(response.EnmuHttptatus.RequestFail, "获取分片失败", nil))
	}

	bucketId := ctx.GetHeader(aws.OssHeaders.BucketId.ToString())
	directoryId := ctx.GetHeader(aws.OssHeaders.DirectoryId.ToString())
	storageId := ctx.GetHeader(aws.OssHeaders.StorageId.ToString())
	uploadId := ctx.GetHeader(aws.OssHeaders.UploadId.ToString())

	contentLength := ctx.Query("content_length")
	partNumber := ctx.Query("part_number")

	file, err := partBytes.Open()
	if err != nil {
		logx.GetLogger("SH").Errorf("Handler|MultipartUploadFile|OpenPartBytesError|%v", err)
		ctx.JSON(http.StatusBadRequest, response.NewResult(response.EnmuHttptatus.RequestFail, "打开分片失败", nil))
	}
	defer file.Close()

	var multipartUploadInfo aws.MultipartUploadInfo
	length, _ := strconv.Atoi(contentLength)
	number, _ := strconv.Atoi(partNumber)
	multipartUploadInfo.ContentLenght = int64(length)
	multipartUploadInfo.PartNumber = int32(number)

	client, err := aws.GetS3Client(storageId)
	if err != nil {
		logx.GetLogger("SH").Errorf("Handler|MultipartUploadFile|GetS3ClientError|%v", err)
		ctx.JSON(http.StatusBadRequest, response.NewResult(response.EnmuHttptatus.RequestFail, "获取s3客户端失败", nil))
	}

	PartEtag, PartNumber, err := aws.MultipartUpload(
		multipartUploadInfo, file, client.S3Client,
		aws.RegionInfo{
			BucketId:    bucketId,
			DirectoryId: directoryId,
			StorageId:   storageId,
		},
		uploadInfo, uploadId)
	if err != nil {
		logx.GetLogger("SH").Errorf("Handler|MultipartUploadFile|MultipartUploadError|%v", err)
		ctx.JSON(http.StatusBadRequest, response.NewResult(response.EnmuHttptatus.RequestFail, "分片上传失败", map[string]int32{
			"part_number": PartNumber,
		}))
		ctx.Abort()
	}

	part := types.CompletedPart{ETag: s3aws.String(PartEtag), PartNumber: s3aws.Int32(PartNumber)}
	rawData, err := json.Marshal(part)
	if err != nil {
		logx.GetLogger("SH").Errorf("Handler|MultipartUploadFile|MarshalError|%v", err)
		panic(err)
	}

	// 存入redis
	err = database.RDB.Set(ctx, fmt.Sprintf(database.Redis_Multipart_Upload_Key, uploadId, PartNumber), rawData, 0).Err()
	if err != nil {
		logx.GetLogger("SH").Errorf("Handler|MultipartUploadFile|SetMultipartUploadInfoToRedisError|%v", err)
		panic(err)
	}
	ctx.JSON(http.StatusOK, response.NewResult(response.EnmuHttptatus.RequestSuccess, "分片上传成功", map[string]int32{
		"part_number": PartNumber,
	}))
}
