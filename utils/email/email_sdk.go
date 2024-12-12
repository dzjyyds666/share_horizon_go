package email

import (
	"ShareHorizon/config"
	"ShareHorizon/utils/log/logx"
	"gopkg.in/gomail.v2"
)

func SendEmail(to string, subject string, body string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", m.FormatAddress(config.GlobalConfig.Email.From, config.GlobalConfig.Email.Alias))
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)

	logx.GetLogger("logx").Infof("邮件配置: %v", config.GlobalConfig.Email)

	d := gomail.NewDialer(
		config.GlobalConfig.Email.Host,
		config.GlobalConfig.Email.Port,
		config.GlobalConfig.Email.User,
		config.GlobalConfig.Email.Password)

	if err := d.DialAndSend(m); err != nil {
		logx.GetLogger("logx").Errorf("发送邮件失败: %v", err)
		return err
	}

	logx.GetLogger("logx").Info("发送邮件成功")
	return nil
}
