package aws

import (
	"ShareHorizon/config"
	"ShareHorizon/utils/log/logx"
	"context"
	"crypto/md5"
	"encoding/base64"
	"errors"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/gabriel-vasile/mimetype"
	"io"
	"os"
	"strconv"
	"strings"
)

type UploadInfo struct {
	//AliasName     *string
	//Bucket        *string
	Fid           *string
	ContentLength *int64
	ContentMd5    *string
	ContentType   *string
	Filename      *string
	Key           *string
}

//func (ui *UploadInfo) WithAliasName(aliasName string) *UploadInfo {
//	if aliasName != "" {
//		ui.AliasName = aws.String(aliasName)
//	} else {
//		logx.GetLogger("SH").Error("OSS-sdk|AliasName can not be empty")
//	}
//	return ui
//}
//
//func (ui *UploadInfo) WithBucket(bucket string) *UploadInfo {
//	if bucket != "" {
//		ui.Bucket = aws.String(bucket)
//	} else {
//		logx.GetLogger("SH").Error("OSS-sdk|Bucket can not be empty")
//	}
//	return ui
//}

func (ui *UploadInfo) WithKey(keys ...string) *UploadInfo {
	join := strings.Join(keys, "/")
	join = join + "/" + aws.ToString(ui.Filename)
	ui.Key = aws.String(join)
	return ui
}

func (ui *UploadInfo) WithFid(fid string) *UploadInfo {
	if fid != "" {
		ui.Fid = aws.String(fid)
	} else {
		logx.GetLogger("SH").Error("OSS-sdk|fid can not be empty")
	}
	return ui
}

func (ui *UploadInfo) WithContentLength(contentLength int64) *UploadInfo {
	if contentLength != 0 {
		ui.ContentLength = aws.Int64(contentLength)
	} else {
		logx.GetLogger("SH").Error("OSS-sdk|contentLength can not be 0")
	}
	return ui
}

func (ui *UploadInfo) WithContentMd5(contentMd5 string) *UploadInfo {
	if contentMd5 != "" {
		ui.ContentMd5 = aws.String(contentMd5)
	} else {
		logx.GetLogger("SH").Error("OSS-sdk|contentMd5 can not be empty")
	}
	return ui
}

func (ui *UploadInfo) WithContentType(contentType string) *UploadInfo {
	if contentType != "" {
		ui.ContentType = aws.String(contentType)
	} else {
		logx.GetLogger("SH").Error("OSS-sdk|contentType can not be empty")
	}
	return ui
}

func (ui *UploadInfo) WithFilename(filename string) *UploadInfo {
	if filename != "" {
		ui.Filename = aws.String(filename)
	} else {
		logx.GetLogger("SH").Error("OSS-sdk|filename can not be empty")
	}
	return ui
}

func GetUploadInfoFromStream(reader io.ReadSeeker, fileName string) (*UploadInfo, error) {
	head := make([]byte, 512)
	_, err := reader.Read(head)
	if err != nil {
		logx.GetLogger("SH").Errorf("OSS-sdk|ReadHeadError|%v", err)
		return nil, err
	}
	contentType := mimetype.Detect(head)

	var contentLength int64
	reader.Seek(0, io.SeekStart)
	buffer := make([]byte, 1024*1024*5)
	hash := md5.New()
	for {
		n, err := reader.Read(buffer)
		if err != nil && err != io.EOF {
			logx.GetLogger("SH").Errorf("OSS-sdk|GetContentType|ReadFileError|%v", err)
			return nil, err
		}
		if n == 0 {
			break
		}
		contentLength += int64(n)
		_, err = hash.Write(buffer[0:n])
		if err != nil {
			logx.GetLogger("SH").Errorf("OSS-sdk|GetContentMd5|WriteHashError|%v", err)
			return nil, err
		}
	}
	hashBytes := hash.Sum(nil)
	hashStr := base64.StdEncoding.EncodeToString(hashBytes)
	return &UploadInfo{
		ContentLength: aws.Int64(contentLength),
		ContentMd5:    aws.String(hashStr),
		ContentType:   aws.String(contentType.String()),
		Filename:      aws.String(fileName),
	}, nil

}

func GetUploadInfoFromLocal(path string) (*UploadInfo, error) {
	ui := &UploadInfo{}
	file, err := os.Open(path)
	defer file.Close()
	if err != nil {
		logx.GetLogger("SH").Errorf("OSS-sdk|OpenFileError|%v", err)
		return nil, err
	}
	stat, _ := file.Stat()
	ui.ContentLength = aws.Int64(stat.Size())
	ui.Filename = aws.String(stat.Name())

	// 读取文件头以确定内容类型
	bytes := make([]byte, 512)
	_, err = io.ReadFull(file, bytes)
	if err != nil && err != io.EOF {
		logx.GetLogger("SH").Errorf("OSS-sdk|GetContentType|ReadFileError|%v", err)
		return nil, err
	}

	contentType := mimetype.Detect(bytes)
	ui.ContentType = aws.String(contentType.String())

	// 重置文件指针到文件开头
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		logx.GetLogger("SH").Errorf("OSS-sdk|SeekFileError|%v", err)
		return nil, err
	}

	hash := md5.New()
	buffer := make([]byte, 1024*1024)
	for {
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			logx.GetLogger("SH").Errorf("OSS-sdk|GetContentMd5|ReadFileError|%v", err)
			return nil, err
		}
		if n == 0 {
			break
		}
		_, err = hash.Write(buffer[0:n])
		if err != nil {
			logx.GetLogger("SH").Errorf("OSS-sdk|GetContentMd5|WriteHashError|%v", err)
			return nil, err
		}
	}
	hashBytes := hash.Sum(nil)
	hashStr := base64.StdEncoding.EncodeToString(hashBytes)
	ui.ContentMd5 = aws.String(hashStr)
	return ui, nil
}

type MultipartUploadInfo struct {
	PartNumber    int32
	ContentLenght int64
}

func GetS3Client(aliasName string) (*config.S3RegionConfig, error) {
	for _, S3config := range config.S3GlobalConfig {
		if S3config.AliasName == aliasName {
			return &S3config, nil
		}
	}
	return &config.S3RegionConfig{}, errors.New("No such S3 config")
}

func PutFile(uploadInfo *UploadInfo, reader io.Reader, aliasName, bucket string) (bool, error) {

	S3Config, err2 := GetS3Client(aliasName)
	if err2 != nil {
		logx.GetLogger("SH").Errorf("OSS-sdk|GetS3Client|%v", err2)
		return false, err2
	}

	if !bucketIsExist(bucket, S3Config.Buckets) {
		logx.GetLogger("SH").Infof("Config|LoadS3Config|BucketNotExist|%s", bucket)
		return false, AwsErrorEnmu.BucketNotExist
	}

	_, err := S3Config.S3Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Body:          reader,
		Bucket:        aws.String(bucket),
		ContentLength: uploadInfo.ContentLength,
		ContentMD5:    uploadInfo.ContentMd5,
		ContentType:   uploadInfo.ContentType,
		Key:           uploadInfo.Key,
		Metadata: map[string]string{
			"fid":            aws.ToString(uploadInfo.Fid),
			"Content-Length": strconv.FormatInt(aws.ToInt64(uploadInfo.ContentLength), 10),
			"Content-MD5":    aws.ToString(uploadInfo.ContentMd5),
		},
	})

	if err != nil {
		logx.GetLogger("SH").Errorf("OSS-sdk|PutObjectError|%v", err)
		return false, AwsErrorEnmu.PutObjetFail
	}
	return true, nil
}

func bucketIsExist(bucketName string, buckets []string) bool {
	for _, bucket := range buckets {
		if bucket == bucketName {
			return true
		}
	}
	return false
}

type InitMultipartFileInfo struct {
	Bucket      string `json:"bucket"`
	DirectoryId string `json:"directory_id"`
	StorageId   string `json:"storage_id"`
	PartitionId string `json:"partition_id"`

	Fid           string `json:"fid"`
	FileName      string `json:"file_name"`
	ContentLength int64  `json:"content_length"`
	ContentType   string `json:"content_type"`
	ContentMd5    string `json:"content_md5"`
	Key           string `json:"key"`

	UserId string `json:"user_id"`

	UploadId string `json:"upload_id"`
}

// 初始化上传
func InitMultUpload(bucket string, client *s3.Client, uploadInfo *UploadInfo) (*s3.CreateMultipartUploadOutput, error) {

	multipartUploadInput := &s3.CreateMultipartUploadInput{
		Bucket:      aws.String(bucket),
		Key:         uploadInfo.Key,
		ContentType: uploadInfo.ContentType,
		Metadata: map[string]string{
			"fid":            aws.ToString(uploadInfo.Fid),
			"Content-Length": strconv.FormatInt(aws.ToInt64(uploadInfo.ContentLength), 10),
			"Content-MD5":    aws.ToString(uploadInfo.ContentMd5),
		},
	}

	ouput, err := client.CreateMultipartUpload(context.TODO(), multipartUploadInput)
	if err != nil {
		logx.GetLogger("SH").Errorf("OSS-sdk|CreateMultipartUpload|%v", err)
		return nil, err
	}

	return ouput, nil
}

// 分片上传
func MultipartUpload(uploadInfo MultipartUploadInfo, r io.Reader, completedParts []*types.CompletedPart, client *s3.Client, info InitMultipartFileInfo) (int32, error) {
	part, err := client.UploadPart(context.TODO(), &s3.UploadPartInput{
		Bucket:        aws.String(info.Bucket),
		Key:           aws.String(info.Key),
		UploadId:      aws.String(info.UploadId),
		Body:          r,
		PartNumber:    aws.Int32(uploadInfo.partNumber),
		ContentLength: aws.Int64(uploadInfo.ContentLenght),
	})
	if err != nil {
		logx.GetLogger("SH").Errorf("OSS-sdk|UploadPart|%v", err)
		return uploadInfo.partNumber, err
	}

	completedParts = append(completedParts, &types.CompletedPart{
		ETag:       part.ETag,
		PartNumber: aws.Int32(uploadInfo.partNumber),
	})
	logx.GetLogger("SH").Infof("OSS-sdk|UploadPart:%v|SUCC", uploadInfo.partNumber)
	return uploadInfo.partNumber, nil
}

// 完成分片上传
func CompleteMultipartUpload(uploadInfo MultipartUploadInfo, completedParts []*types.CompletedPart, client *s3.Client) error {
	var parts []types.CompletedPart
	for _, part := range completedParts {
		parts = append(parts, *part)
	}

	_, err := client.CompleteMultipartUpload(context.TODO(), &s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(uploadInfo.Bucket),
		Key:      aws.String(uploadInfo.Key),
		UploadId: aws.String(uploadInfo.UploadId),
		MultipartUpload: &types.CompletedMultipartUpload{
			Parts: parts,
		},
	})
	if err != nil {
		logx.GetLogger("SH").Errorf("OSS-sdk|CompleteMultipartUpload|%v", err)
		return err
	}
	return nil
}

// 中断上传
func AbortMultipartUpload(uploadInfo MultipartUploadInfo, client *s3.Client) error {
	_, err := client.AbortMultipartUpload(context.TODO(), &s3.AbortMultipartUploadInput{
		Bucket:   aws.String(uploadInfo.Bucket),
		Key:      aws.String(uploadInfo.Key),
		UploadId: aws.String(uploadInfo.UploadId),
	})
	if err != nil {
		logx.GetLogger("SH").Errorf("OSS-sdk|AbortMultipartUploadError|%v", err)
		return err
	}
	return nil
}
