package config

import (
	"ShareHorizon/utils/log/logx"
	"testing"
)

func TestLoggingConfig(t *testing.T) {
	config := LoadS3Config("s3_config.toml")
	logx.GetLogger("SH").Infof("Config|LoadS3Config|%v", config)
}
