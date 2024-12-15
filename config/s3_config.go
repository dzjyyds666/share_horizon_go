package config

import (
	"ShareHorizon/utils/log/logx"
	"context"
	"github.com/BurntSushi/toml"
	"github.com/aws/aws-sdk-go-v2/aws"
	S3config "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/mitchellh/mapstructure"
	"net/http"
	"os"
	"time"
)

type S3RegionConfig struct {
	Endpoint  string   `mapstructure:"url"`
	AliasName string   `mapstructure:"alias_name"`
	AccessId  string   `mapstructure:"accessId"`
	SecretKey string   `mapstructure:"secretKey"`
	IamRoleid string   `mapstructure:"iamRoleid"`
	Region    string   `mapstructure:"region"`
	Buckets   []string `mapstructure:"buckets"`

	S3Client *s3.Client
	hcli     *http.Client
}

type S3Config struct {
	Regions map[string]S3RegionConfig `mapstructure:"regions"`
}

var S3GlobalConfig []S3RegionConfig

func init() {
	S3GlobalConfig = LoadS3Config("./config/s3_config.toml")

	// 创建 S3 客户端
	for i, regionConfig := range S3GlobalConfig {
		config, err := S3config.LoadDefaultConfig(context.TODO(),
			S3config.WithRegion(regionConfig.Region),
			S3config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(regionConfig.AccessId, regionConfig.SecretKey, "")),
			S3config.WithBaseEndpoint(regionConfig.Endpoint),
		)
		if err != nil {
			logx.GetLogger("SH").Errorf("Config|InitS3Config|CreateClientError|%v", err)
			os.Exit(1)
		}

		S3GlobalConfig[i].S3Client = s3.NewFromConfig(config, func(options *s3.Options) {
			options.HTTPClient = regionConfig.hcli
			options.UsePathStyle = true
		})

		for _, bucket := range regionConfig.Buckets {
			// 判断bucket是否存在,不存在就创建
			bucketErr := exitOrCreateBucket(S3GlobalConfig[i].S3Client, bucket, regionConfig.Region)
			if bucketErr != nil {
				logx.GetLogger("SH").Errorf("Config|InitS3Config|CreateBucketError|%v", bucketErr)
				os.Exit(1)
			}
		}
	}
	logx.GetLogger("SH").Infof("S3Config:%v", S3GlobalConfig)
	logx.GetLogger("SH").Info("Config|InitS3Config|SUCC")
}

func LoadS3Config(filePath string) []S3RegionConfig {
	data, err := os.ReadFile(filePath)
	if err != nil {
		logx.GetLogger("SH").Errorf("Config|LoadS3Config|ReadError|%v", err)
		os.Exit(1)
	}

	// 解析TOML格式的配置文件
	var rawConfig map[string]interface{}
	if err = toml.Unmarshal(data, &rawConfig); err != nil {
		logx.GetLogger("SH").Errorf("Config|LoadS3Config|UnmarshalError|%v", err)
		os.Exit(1)
	}

	// 获取所有区域的配置
	regionsMap, ok := rawConfig["regions"].(map[string]interface{})
	if !ok {
		logx.GetLogger("SH").Errorf("Config|LoadS3Config|InvalidConfig|%v", err)
		os.Exit(1)
	}

	var s3Configs []S3RegionConfig

	for _, regionConfig := range regionsMap {
		var s3RegionConfig S3RegionConfig
		if err := mapstructure.Decode(regionConfig, &s3RegionConfig); err != nil {
			os.Exit(1)
		}
		s3RegionConfig.hcli = &http.Client{
			Timeout: 30 * time.Second,
		}
		s3Configs = append(s3Configs, s3RegionConfig)
	}
	return s3Configs
}

func exitOrCreateBucket(S3Client *s3.Client, bucket, region string) error {
	_, err := S3Client.HeadBucket(context.TODO(), &s3.HeadBucketInput{Bucket: aws.String(bucket)})
	if err != nil {
		// 创建桶
		_, createrr := S3Client.CreateBucket(context.TODO(), &s3.CreateBucketInput{
			Bucket: aws.String(bucket),
			CreateBucketConfiguration: &types.CreateBucketConfiguration{
				LocationConstraint: types.BucketLocationConstraint(region),
			},
			ACL: types.BucketCannedACLPublicRead,
		})
		if createrr != nil {
			logx.GetLogger("SH").Errorf("Bucket|CreateBucket|Error|%v", createrr)
			return createrr
		} else {
			return nil
		}
	} else {
		return nil
	}
}
