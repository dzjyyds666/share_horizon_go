package handlers

import (
	"ShareHorizon/database"
	"ShareHorizon/models"
	"ShareHorizon/utils/aws"
	"ShareHorizon/utils/log/logx"
	"ShareHorizon/utils/response"
	"context"
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

// @Summary 申请上传文件
// @Tags 文件上传
// @Accept json
// @Produce json
// @Param X-Oss-StorageId header string true "存储ID"
// @Param X-Oss-BucketId header string true "Bucket名称"
// @Param X-Oss-DirectoryId header string true "目录ID"
// @Success 200 {object} response.Result "登录成功"
// @Router /upload/applayUpload [post]
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
	err = database.RDB[0].Set(ctx, fmt.Sprintf(database.Redis_Applay_Upload_Fid, fid), marshal, time.Minute*5).Err()
	if err != nil {
		logx.GetLogger("SH").Errorf("Handler|ApplayUpload|SetRedisError|%v", err)
		panic(err)
	}
	ctx.JSON(http.StatusOK, response.NewResult(response.EnmuHttptatus.RequestSuccess, "申请上传成功", fid))
}

// @Summary 上传文件
// @Tags 文件上传
// @Description 直接上传文件
// @Accept multipart/form-data
// @Produce json
// @Param fid path string true "fid"
// @Param file_name formData string true "file_name"
// @Param file formData file true "file"
// @Router /upload/putFile/{fid} [post]
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

	fid := ctx.Param("fid")

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
	value, err := database.RDB[0].Get(ctx, fmt.Sprintf(database.Redis_Applay_Upload_Fid, fid)).Result()
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

// @Summary 初始化分片上传
// @Tags 文件上传
// @Description 初始化分片上传
// @Accept json
// @Produce json
// @Param uploadInfo body aws.UploadInfo true "uploadInfo"
// @Router /upload/initMultipartUpload [post]
func InitMultipartFile(ctx *gin.Context) {
	var uploadInfo aws.UploadInfo
	err := ctx.ShouldBindJSON(&uploadInfo)
	if err != nil {
		logx.GetLogger("SH").Errorf("Handler|InitMultipartFile|ParamsError|%v", err)
		ctx.JSON(http.StatusBadRequest, response.NewResult(response.EnmuHttptatus.ParamError, "参数错误", nil))
		ctx.Abort()
	}

	// 查询redis是否存在fid信息
	result, err := database.RDB[0].Get(ctx, fmt.Sprintf(database.Redis_Applay_Upload_Fid, uploadInfo.Fid)).Result()
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

	// 把分片信息存入redis
	var partNumber int64
	if uploadInfo.ContentLength < uploadInfo.PartSize {
		partNumber = 1
	} else if (uploadInfo.ContentLength % uploadInfo.PartSize) > 0 {
		partNumber = uploadInfo.ContentLength/uploadInfo.PartSize + 1
	} else {
		partNumber = uploadInfo.ContentLength / uploadInfo.PartSize
	}
	err = database.RDB[0].Set(ctx, fmt.Sprintf(database.Redis_Multipart_Upload_Key, "partNumber", uploadInfo.Fid), strconv.Itoa(int(partNumber)), 0).Err()
	if err != nil {
		logx.GetLogger("SH").Errorf("Handler|InitMultipartFile|SetRedisError|%v", err)
		panic(err)
	}

	ctx.JSON(http.StatusOK, response.NewResult(response.EnmuHttptatus.RequestSuccess, "初始化分片上传成功", map[string]string{
		"upload_id":  s3aws.ToString(upload.UploadId),
		"upload_key": uploadInfo.Key,
	}))
}

// @Summary 分片上传
// @Tags 文件上传
// @Description 分片上传
// @Accept multipart/form-data
// @Produce json
// @Param fid path string true "fid"
// @Param part_bytes formData file true "part_bytes"
// @Param X-Oss-UploadId header string true "upload_id"
// @Param X-Oss-UploadKey header string true "upload_key"
// @Param content_length query string true "content_length"
// @Param part_number query string true "part_number"
// @Router /upload/multipart/{fid} [post]
func MultipartUploadFile(ctx *gin.Context) {
	key := ctx.GetHeader(aws.OssHeaders.UploadKey.ToString())
	uploadId := ctx.GetHeader(aws.OssHeaders.UploadId.ToString())

	fid := ctx.Param("fid")

	contentLength := ctx.Query("content_length")
	partNumber := ctx.Query("part_number")

	// 查询是否fid是否存在
	err := database.RDB[0].Get(ctx, fmt.Sprintf(database.Redis_Applay_Upload_Fid, fid)).Err()
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

	// 查询redis是否存在fid信息
	result, err := database.RDB[0].Get(ctx, fmt.Sprintf(database.Redis_Applay_Upload_Fid, fid)).Result()
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

	client, err := aws.GetS3Client(applayUpload.StorageId)
	if err != nil {
		logx.GetLogger("SH").Errorf("Handler|MultipartUploadFile|GetS3ClientError|%v", err)
		ctx.JSON(http.StatusBadRequest, response.NewResult(response.EnmuHttptatus.RequestFail, "获取s3客户端失败", nil))
	}

	PartEtag, PartNumber, err := aws.MultipartUpload(
		multipartUploadInfo, file, client.S3Client,
		aws.RegionInfo{
			BucketId:    applayUpload.Bucket,
			DirectoryId: applayUpload.DirectoryId,
			StorageId:   applayUpload.StorageId,
		},
		key, uploadId)
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
	err = database.RDB[0].Set(ctx, fmt.Sprintf(database.Redis_Multipart_Upload_Key, uploadId, PartNumber), rawData, 0).Err()
	if err != nil {
		logx.GetLogger("SH").Errorf("Handler|MultipartUploadFile|SetMultipartUploadInfoToRedisError|%v", err)
		panic(err)
	}
	ctx.JSON(http.StatusOK, response.NewResult(response.EnmuHttptatus.RequestSuccess, "分片上传成功", map[string]int32{
		"part_number": PartNumber,
	}))
}

// CompleteMultipartUpload
// @Summary 完成分片上传
// @Tags 文件上传
// @Description 完成分片上传
// @Prodecu json
// @Param X-Oss-UploadId header string true "upload_id"
// @Param X-Oss-UploadKey header string true "upload_key"
// @Param fid path string true "fid"
// @Router /upload/multipart/complete/{fid} [post]
func CompleteMultipartUpload(ctx *gin.Context) {
	// 需要completedParts []*types.CompletedPart, client *s3.Client, key, bucket, uploadId string
	uploadId := ctx.GetHeader(aws.OssHeaders.UploadId.ToString())
	//bucketId := ctx.GetHeader(aws.OssHeaders.BucketId.ToString())
	//storageId := ctx.GetHeader(aws.OssHeaders.StorageId.ToString())

	fid := ctx.Param("fid")
	// 查询redis是否存在fid信息
	result, err := database.RDB[0].Get(ctx, fmt.Sprintf(database.Redis_Applay_Upload_Fid, fid)).Result()
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

	key := ctx.GetHeader(aws.OssHeaders.UploadKey.ToString())
	client, err := aws.GetS3Client(applayUpload.StorageId)
	if err != nil {
		logx.GetLogger("SH").Errorf("Handler|CompleteMultipartUpload|GetS3ClientError|%v", err)
		ctx.JSON(http.StatusBadRequest, response.NewResult(response.EnmuHttptatus.RequestFail, "获取s3客户端失败", nil))
	}

	// 从redis中获取到上传信息
	var completeParts []types.CompletedPart
	var cursor uint64
	for {
		var keys []string
		var redisErr error
		keys, cursor, redisErr = database.RDB[0].Scan(ctx, cursor, fmt.Sprintf(database.Redis_Multipart_Upload_Key, uploadId, "*"), 0).Result()
		if redisErr != nil {
			logx.GetLogger("SH").Errorf("Handler|CompleteMultipartUpload|ScanError|%v", redisErr)
			panic(redisErr)
		}

		for _, key := range keys {
			value, getError := database.RDB[0].Get(ctx, key).Result()
			if getError != nil {
				logx.GetLogger("SH").Errorf("Handler|CompleteMultipartUpload|GetError|%v", getError)
				panic(getError)
			}

			var completePart types.CompletedPart
			jsonErr := json.Unmarshal([]byte(value), &completePart)
			if jsonErr != nil {
				logx.GetLogger("SH").Errorf("Handler|CompleteMultipartUpload|JsonError|%v", jsonErr)
				panic(jsonErr)
			}
			completeParts = append(completeParts, completePart)
		}
		if cursor == 0 {
			break
		}
	}

	// 从redis中获取到文件分片数目
	result, err = database.RDB[0].Get(ctx, fmt.Sprintf(database.Redis_Multipart_Upload_Key, "partNumber", uploadId)).Result()
	if err != nil {
		logx.GetLogger("SH").Errorf("Handler|CompleteMultipartUpload|GetPartNumberError|%v", err)
		panic(err)
	}

	if partnumber, _ := strconv.Atoi(result); len(completeParts) != partnumber {
		logx.GetLogger("SH").Errorf("partNumber Is Not Enough")
		ctx.JSON(http.StatusBadRequest, response.NewResult(response.EnmuHttptatus.RequestFail, "上传分片数目不够,完成失败", nil))
		ctx.Abort()
	}

	// 将 completeParts 转换为 []*types.CompletedPart
	var completedPartsPtr []*types.CompletedPart
	for _, part := range completeParts {
		completedPartsPtr = append(completedPartsPtr, &part)
	}

	err = aws.CompleteMultipartUpload(completedPartsPtr, client.S3Client, key, applayUpload.Bucket, uploadId)
	if err != nil {
		logx.GetLogger("SH").Errorf("Handler|CompleteMultipartUpload|CompleteMultipartUploadError|%v", err)
		ctx.JSON(http.StatusBadRequest, response.NewResult(response.EnmuHttptatus.RequestFail, "完成分片上传失败", nil))
	}

	defer func() {
		// 调用携程删除上传文件信息
		go deleteRedisKey(database.Redis_Multipart_Upload_Key, uploadId)
	}()

	ctx.JSON(http.StatusOK, response.NewResult(response.EnmuHttptatus.RequestSuccess, "完成分片上传成功", nil))
}

// 删除redis中上传完毕的信息
func deleteRedisKey(prefix, key string) {
	key = fmt.Sprintf(prefix, key, "*")
	var cursor uint64
	var keys []string
	var err error
	for {
		keys, cursor, err = database.RDB[0].Scan(context.TODO(), cursor, key, 0).Result()
		if err != nil {
			logx.GetLogger("SH").Errorf("Handler|CompleteMultipartUpload|ScanError|%v", err)
			return
		}

		for _, k := range keys {
			err = database.RDB[0].Del(context.TODO(), k).Err()
			if err != nil {
				logx.GetLogger("SH").Errorf("Handler|CompleteMultipartUpload|DelError|%v", err)
				return
			}
		}
		if cursor == 0 {
			break
		}
	}
}

// @Summary 中断分片上传
// @Description 中断分片上传
// @Tags 文件上传
// @Param X-Oss-UploadId header string true "upload_id"
// @Param fid path string true "fid"
// @Router /upload/multipart/abort/{fid} [post]
func AbortMultipartUpload(ctx *gin.Context) {

	uploadId := ctx.GetHeader(aws.OssHeaders.UploadId.ToString())

	//bucketId := ctx.GetHeader(aws.OssHeaders.BucketId.ToString())
	//storageId := ctx.GetHeader(aws.OssHeaders.StorageId.ToString())
	fid := ctx.Param("fid")
	// 查询redis是否存在fid信息
	result, err := database.RDB[0].Get(ctx, fmt.Sprintf(database.Redis_Applay_Upload_Fid, fid)).Result()
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
	key := ctx.GetHeader(aws.OssHeaders.UploadKey.ToString())

	client, err := aws.GetS3Client(applayUpload.StorageId)
	if err != nil {
		logx.GetLogger("SH").Errorf("Handler|AbortMultipartUpload|GetS3ClientError|%v", err)
		ctx.JSON(http.StatusBadRequest, response.NewResult(response.EnmuHttptatus.RequestFail, "获取s3客户端失败", nil))
	}
	err = aws.AbortMultipartUpload(client.S3Client, key, applayUpload.Bucket, uploadId)
	if err != nil {
		logx.GetLogger("SH").Errorf("Handler|AbortMultipartUpload|AbortMultipartUploadError|%v", err)
		ctx.JSON(http.StatusBadRequest, response.NewResult(response.EnmuHttptatus.RequestFail, "中断分片上传失败", nil))
	}

	defer func() {
		go deleteRedisKey(database.Redis_Multipart_Upload_Key, uploadId)
	}()
	ctx.JSON(http.StatusOK, response.NewResult(response.EnmuHttptatus.RequestSuccess, "中断分片上传成功", nil))
}
