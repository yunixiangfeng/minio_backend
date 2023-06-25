package test

import (
	"core/define"
	"crypto/tls"
	"net/smtp"
	"testing"

	"github.com/jordan-wright/email"
)

func TestSendMail(t *testing.T) {
	e := email.NewEmail()
	e.From = "云盘 <535504958@qq.com>"
	e.To = []string{"the.wu@qq.com"}
	e.Subject = "验证码发送测试"
	e.HTML = []byte("<pre style=\"font-family:Helvetica,arial,sans-serif;font-size:13px;color:#747474;text-align:left;line-height:18px\">欢迎使用云盘，您的验证码为：<span style=\"font-size:block\">123456</span></pre>")
	err := e.SendWithTLS("smtp.163.com:465", smtp.PlainAuth("", "535504958@qq.com", define.MailPassword, "smtp.163.com"),
		&tls.Config{InsecureSkipVerify: true, ServerName: "smtp.163.com"})
	if err != nil {
		t.Fatal(err)
	}
}
