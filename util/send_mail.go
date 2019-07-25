package util

import (
	"crypto/tls"
	"github.com/go-gomail/gomail"
	"strings"
)

var serverHost, fromEmail, fromPasswd string
var serverPort int
var m *gomail.Message

type EmailParam struct {
	// ServerHost 邮箱服务器地址
	ServerHost string
	// ServerPort 邮箱服务器端口
	ServerPort int
	// FromEmail　发件人邮箱地址
	FromEmail string
	// FromPasswd 这个密码是smtp服务的密码，在邮箱中打开pop3/smtp服务，使用授权码
	FromPassword string
	// Toers 接收者邮件，如有多个，则以英文逗号隔开，不能为空
	Toers string
	// CCers 抄送者邮件，如有多个，则以英文逗号隔开，可以为空
	CCers string
}

func InitNewUser() *EmailParam {
	// 可以在这里写死
	return &EmailParam {
		FromEmail:  "",
		FromPassword: "",
		ServerHost: "",
		ServerPort: 0,
		Toers:      "",
	}
}

func InitEmail(ep *EmailParam) {
	toers := []string{}

	serverHost = ep.ServerHost
	serverPort = ep.ServerPort
	fromEmail = ep.FromEmail
	fromPasswd = ep.FromPassword

	m = gomail.NewMessage()

	if len(ep.Toers) == 0 {
		return
	}

	for _, tmp := range strings.Split(ep.Toers, ",") {
		toers = append(toers, strings.TrimSpace(tmp))
	}

	// 收件人可以有多个，故用此方式
	m.SetHeader("To", toers...)

	//抄送列表
	if len(ep.CCers) != 0 {
		for _, tmp := range strings.Split(ep.CCers, ",") {
			toers = append(toers, strings.TrimSpace(tmp))
		}
		m.SetHeader("Cc", toers...)
	}

	// 发件人
	// 第三个参数为发件人别名，如"Tom"，可以为空（此时则为邮箱名称）
	m.SetAddressHeader("From", fromEmail, "")
}

func SendEmail(subject, body string, attachment string) error {
	// 主题
	m.SetHeader("Subject", subject)
	// 正文
	m.SetBody("text/html", body)
	//m.SetBody("text/plain", body)
	if attachment != "" {
		m.Attach(attachment)
	}

	d := gomail.NewPlainDialer(serverHost, serverPort, fromEmail, fromPasswd)
	d.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	// 发送
	err := d.DialAndSend(m)
	if err != nil {
		return err
	}

	return nil
}