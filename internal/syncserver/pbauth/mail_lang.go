package pbauth

import (
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

const (
	codeSubjectZh = core.EmailPlaceholderOTP + " 是你的 " + appName + " 验证码"
	codeBodyZh    = `<div style="font-family:-apple-system,Segoe UI,PingFang SC,Helvetica,Arial,sans-serif;color:#1d1d1f;max-width:420px">
<p>你的验证码：</p>
<p style="font-family:ui-monospace,SFMono-Regular,Menlo,Consolas,monospace;font-size:34px;font-weight:600;letter-spacing:8px;margin:16px 0">` + core.EmailPlaceholderOTP + `</p>
<p>10 分钟内有效。如果不是你在操作，忽略这封邮件即可；没有验证码谁也登录不了。</p>
<p style="color:#6e6e73">&mdash; ` + appName + ` &middot; todo.sri-ai.xyz</p>
</div>`
)

// localizeCodeMail writes the code mail in the language the user's account was made in. English is
// the collection's template and the default (an account made by Google, or before `lang` existed,
// has none). The message is changed before PocketBase sends it; the send itself, and the sentTo
// mark that lets the code verify the account, stay PocketBase's.
func localizeCodeMail(e *core.MailerRecordEvent) error {
	if e.Record != nil && strings.EqualFold(e.Record.GetString("lang"), langChinese) {
		code, _ := e.Meta["password"].(string)
		e.Message.Subject = strings.ReplaceAll(codeSubjectZh, core.EmailPlaceholderOTP, code)
		e.Message.HTML = strings.ReplaceAll(codeBodyZh, core.EmailPlaceholderOTP, code)
	}
	return e.Next()
}
