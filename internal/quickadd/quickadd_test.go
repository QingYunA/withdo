package quickadd

import (
	"testing"
	"time"
)

var zone = time.FixedZone("CST", 8*3600)

func at(y int, mo time.Month, d, h, mi int) time.Time { return time.Date(y, mo, d, h, mi, 0, 0, zone) }

// wed is Wednesday 2026-10-21 10:00, the "now" of most cases.
var wed = at(2026, 10, 21, 10, 0)

// want is a draft with its times as "2006-01-02 15:04" text, so a table reads at a glance.
type want struct {
	title, due, remind, cat string
	important               bool
}

func show(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02 15:04")
}

func flat(ds []Draft) []want {
	out := make([]want, len(ds))
	for i, d := range ds {
		out[i] = want{d.Title, show(d.Due), show(d.RemindAt), d.Category, d.Important}
	}
	return out
}

var cats = []string{"工作", "生活", "Work", "Home"}

func TestParse(t *testing.T) {
	sun := at(2026, 10, 25, 10, 0)
	tests := []struct {
		name string
		in   string
		now  time.Time
		cats []string
		want []want
	}{
		// plain text
		{"plain", "买牛奶", wed, cats, []want{{title: "买牛奶"}}},
		{"empty", "", wed, cats, nil},
		{"blank", "  \n ", wed, cats, nil},
		{"only a separator", "；", wed, cats, nil},
		{"digits in a title", "买 3 个苹果", wed, cats, []want{{title: "买 3 个苹果"}}},
		{"a bit is not one o'clock", "买一点水果", wed, cats, []want{{title: "买一点水果"}}},
		{"a weight is not a date", "买3.5kg盐", wed, cats, []want{{title: "买3.5kg盐"}}},
		{"nothing but a date keeps its words", "明天", wed, cats, []want{{title: "明天"}}},
		{"spaces collapse", "  明天   交房租  ", wed, cats, []want{{"交房租", "2026-10-22 18:00", "", "", false}}},

		// zh days
		{"tomorrow", "明天交房租", wed, cats, []want{{"交房租", "2026-10-22 18:00", "", "", false}}},
		{"day after", "后天体检", wed, cats, []want{{"体检", "2026-10-23 18:00", "", "", false}}},
		{"two days after", "大后天开会", wed, cats, []want{{"开会", "2026-10-24 18:00", "", "", false}}},
		{"today", "今天写周报", wed, cats, []want{{"写周报", "2026-10-21 18:00", "", "", false}}},
		{"possessive particle goes", "明天的会议", wed, cats, []want{{"会议", "2026-10-22 18:00", "", "", false}}},
		{"weekday", "周五交报告", wed, cats, []want{{"交报告", "2026-10-23 18:00", "", "", false}}},
		{"bare weekday is after today", "星期三交报告", wed, cats, []want{{"交报告", "2026-10-28 18:00", "", "", false}}},
		{"next week weekday", "下周三交方案", wed, cats, []want{{"交方案", "2026-10-28 18:00", "", "", false}}},
		{"this week weekday", "这周五交报告", wed, cats, []want{{"交报告", "2026-10-23 18:00", "", "", false}}},
		{"this week weekday already gone", "这周一交报告", wed, cats, []want{{"交报告", "2026-10-26 18:00", "", "", false}}},
		{"weekday sunday word", "周天爬山", wed, cats, []want{{"爬山", "2026-10-25 18:00", "", "", false}}},
		{"on sunday next monday is tomorrow", "下周一开会", sun, cats, []want{{"开会", "2026-10-26 18:00", "", "", false}}},
		{"on sunday this sunday is today", "这周日写总结", sun, cats, []want{{"写总结", "2026-10-25 18:00", "", "", false}}},
		{"on sunday bare sunday is a week away", "周日写总结", sun, cats, []want{{"写总结", "2026-11-01 18:00", "", "", false}}},
		{"on sunday monday is tomorrow", "周一开会", sun, cats, []want{{"开会", "2026-10-26 18:00", "", "", false}}},
		{"month and day", "10月31日交稿", wed, cats, []want{{"交稿", "2026-10-31 18:00", "", "", false}}},
		{"dotted date", "10.31 交稿", wed, cats, []want{{"交稿", "2026-10-31 18:00", "", "", false}}},
		{"dashed date", "10-31交稿", wed, cats, []want{{"交稿", "2026-10-31 18:00", "", "", false}}},
		{"iso date", "2026-11-02 交稿", wed, cats, []want{{"交稿", "2026-11-02 18:00", "", "", false}}},
		{"a date already gone is next year", "1月5日体检", wed, cats, []want{{"体检", "2027-01-05 18:00", "", "", false}}},
		{"today's date", "10月21日体检", wed, cats, []want{{"体检", "2026-10-21 18:00", "", "", false}}},
		{"day of month still ahead", "31号结算", wed, cats, []want{{"结算", "2026-10-31 18:00", "", "", false}}},
		{"day of month gone is next month", "15号交房租", wed, cats, []want{{"交房租", "2026-11-15 18:00", "", "", false}}},
		{"a metro line is not a day", "坐15号线", wed, cats, []want{{title: "坐15号线"}}},
		{"impossible date stays in the title", "2月30日开会", wed, cats, []want{{title: "2月30日开会"}}},
		{"year end rolls over", "明天交报告", at(2026, 12, 31, 10, 0), cats, []want{{"交报告", "2027-01-01 18:00", "", "", false}}},
		{"month end rolls over", "后天交报告", at(2027, 2, 27, 10, 0), cats, []want{{"交报告", "2027-03-01 18:00", "", "", false}}},

		// zh times
		{"nine o'clock", "明天九点开会", wed, cats, []want{{"开会", "2026-10-22 09:00", "", "", false}}},
		{"digit hour", "明天9点开会", wed, cats, []want{{"开会", "2026-10-22 09:00", "", "", false}}},
		{"colon time", "明天9:00开会", wed, cats, []want{{"开会", "2026-10-22 09:00", "", "", false}}},
		{"afternoon three", "明天下午3点开会", wed, cats, []want{{"开会", "2026-10-22 15:00", "", "", false}}},
		{"evening half past", "明天晚上8点半吃饭", wed, cats, []want{{"吃饭", "2026-10-22 20:30", "", "", false}}},
		{"minutes", "明天下午三点15分开会", wed, cats, []want{{"开会", "2026-10-22 15:15", "", "", false}}},
		{"noon", "明天中午吃饭", wed, cats, []want{{"吃饭", "2026-10-22 12:00", "", "", false}}},
		{"afternoon alone", "明天下午开会", wed, cats, []want{{"开会", "2026-10-22 14:00", "", "", false}}},
		{"evening alone", "明天晚上看电影", wed, cats, []want{{"看电影", "2026-10-22 19:00", "", "", false}}},
		{"tonight", "今晚看球", wed, cats, []want{{"看球", "2026-10-21 20:00", "", "", false}}},
		{"tomorrow morning", "明早跑步", wed, cats, []want{{"跑步", "2026-10-22 09:00", "", "", false}}},
		{"a time with no date is the next one: tomorrow", "九点开会", wed, cats, []want{{"开会", "2026-10-22 09:00", "", "", false}}},
		{"a time with no date is the next one: today", "下午3点开会", wed, cats, []want{{"开会", "2026-10-21 15:00", "", "", false}}},
		{"a small hour with no period is the afternoon", "3点开会", wed, cats, []want{{"开会", "2026-10-21 15:00", "", "", false}}},
		{"an explicit time today may have passed", "今天9点开会", wed, cats, []want{{"开会", "2026-10-21 09:00", "", "", false}}},

		// en
		{"en tomorrow", "call mom tomorrow", wed, cats, []want{{"call mom", "2026-10-22 18:00", "", "", false}}},
		{"en today", "call mom today", wed, cats, []want{{"call mom", "2026-10-21 18:00", "", "", false}}},
		{"en weekday", "pay rent monday", wed, cats, []want{{"pay rent", "2026-10-26 18:00", "", "", false}}},
		{"en weekday on", "pay rent on Friday", wed, cats, []want{{"pay rent", "2026-10-23 18:00", "", "", false}}},
		{"en next weekday", "pay rent next friday", wed, cats, []want{{"pay rent", "2026-10-30 18:00", "", "", false}}},
		{"en this weekday", "pay rent this friday", wed, cats, []want{{"pay rent", "2026-10-23 18:00", "", "", false}}},
		{"en month day", "pay rent Oct 31", wed, cats, []want{{"pay rent", "2026-10-31 18:00", "", "", false}}},
		{"en long month ordinal", "pay rent october 31st", wed, cats, []want{{"pay rent", "2026-10-31 18:00", "", "", false}}},
		{"en day month", "pay rent 31 Oct", wed, cats, []want{{"pay rent", "2026-10-31 18:00", "", "", false}}},
		{"en at hour", "call mom tomorrow at 9", wed, cats, []want{{"call mom", "2026-10-22 09:00", "", "", false}}},
		{"en am", "call mom at 9am", wed, cats, []want{{"call mom", "2026-10-22 09:00", "", "", false}}},
		{"en pm minutes", "meeting 3:30pm", wed, cats, []want{{"meeting", "2026-10-21 15:30", "", "", false}}},
		{"en time before date", "9am tomorrow standup", wed, cats, []want{{"standup", "2026-10-22 09:00", "", "", false}}},
		{"en noon", "lunch at noon", wed, cats, []want{{"lunch", "2026-10-21 12:00", "", "", false}}},
		{"en noon gone", "lunch at noon", at(2026, 10, 21, 13, 0), cats, []want{{"lunch", "2026-10-22 12:00", "", "", false}}},
		{"a deadline beats remind me", "提醒我周五前交报告", wed, cats, []want{{"交报告", "2026-10-23 18:00", "", "", false}}},
		{"en at 7pm", "Meet Sam at 7pm", wed, cats, []want{{"Meet Sam", "2026-10-21 19:00", "", "", false}}},
		{"en tonight", "watch the game tonight", wed, cats, []want{{"watch the game", "2026-10-21 20:00", "", "", false}}},

		// reminders and deadlines
		{"zh remind", "提醒我明天体检", wed, cats, []want{{"体检", "", "2026-10-22 09:00", "", false}}},
		{"zh remind with a time", "提醒我明天下午3点体检", wed, cats, []want{{"体检", "", "2026-10-22 15:00", "", false}}},
		{"zh remind without a date", "提醒我买牛奶", wed, cats, []want{{title: "买牛奶"}}},
		{"en remind", "remind me to call mom tomorrow", wed, cats, []want{{"call mom", "", "2026-10-22 09:00", "", false}}},
		{"en remind at", "remind me to call mom at 5pm", wed, cats, []want{{"call mom", "", "2026-10-21 17:00", "", false}}},
		{"en remind without a date", "remind me to call mom", wed, cats, []want{{title: "call mom"}}},
		{"zh before", "周五前交报告", wed, cats, []want{{"交报告", "2026-10-23 18:00", "", "", false}}},
		{"zh before a time", "周五下午3点前交报告", wed, cats, []want{{"交报告", "2026-10-23 15:00", "", "", false}}},
		{"zh deadline word", "报告截止周五", wed, cats, []want{{"报告", "2026-10-23 18:00", "", "", false}}},
		{"zh before a time today", "下午5点前提交", wed, cats, []want{{"提交", "2026-10-21 17:00", "", "", false}}},
		{"a front desk is not a deadline", "明天前台取快递", wed, cats, []want{{"前台取快递", "2026-10-22 18:00", "", "", false}}},
		{"en by", "submit report by friday", wed, cats, []want{{"submit report", "2026-10-23 18:00", "", "", false}}},
		{"en before", "finish slides before tomorrow 5pm", wed, cats, []want{{"finish slides", "2026-10-22 17:00", "", "", false}}},
		{"en by in a name", "a book by Tolstoy", wed, cats, []want{{title: "a book by Tolstoy"}}},

		// categories
		{"category first", "#工作 写周报", wed, cats, []want{{"写周报", "", "", "工作", false}}},
		{"category last", "写周报 #工作", wed, cats, []want{{"写周报", "", "", "工作", false}}},
		{"category without a space", "买牛奶#生活", wed, cats, []want{{"买牛奶", "", "", "生活", false}}},
		{"category case", "write report #work", wed, cats, []want{{"write report", "", "", "Work", false}}},
		{"unknown category stays", "#unknown thing", wed, cats, []want{{title: "#unknown thing"}}},
		{"a hash in a word stays", "learn C#", wed, cats, []want{{title: "learn C#"}}},
		{"category prefix of a longer word stays", "fix #homepage", wed, cats, []want{{title: "fix #homepage"}}},
		{"no categories", "写周报 #工作", wed, nil, []want{{title: "写周报 #工作"}}},

		// importance
		{"bang", "交房租!", wed, cats, []want{{"交房租", "", "", "", true}}},
		{"full width bang", "交房租！", wed, cats, []want{{"交房租", "", "", "", true}}},
		{"several bangs", "交房租!!", wed, cats, []want{{"交房租", "", "", "", true}}},
		{"important word", "交房租 重要", wed, cats, []want{{"交房租", "", "", "", true}}},
		{"a bang in the middle stays", "wow! really", wed, cats, []want{{title: "wow! really"}}},
		{"everything at once", "#生活 周六大扫除!", wed, cats, []want{{"大扫除", "2026-10-24 18:00", "", "生活", true}}},
		{"bang before a date", "交房租! 明天", wed, cats, []want{{"交房租", "2026-10-22 18:00", "", "", true}}},

		// splitting
		{"zh also", "明天交房租，还有体检", wed, cats, []want{{"交房租", "2026-10-22 18:00", "", "", false}, {title: "体检"}}},
		{"zh semicolon", "买牛奶；交房租", wed, cats, []want{{title: "买牛奶"}, {title: "交房租"}}},
		{"zh then", "写周报然后发邮件", wed, cats, []want{{title: "写周报"}, {title: "发邮件"}}},
		{"en and also", "call mom, and also pay rent", wed, cats, []want{{title: "call mom"}, {title: "pay rent"}}},
		{"en semicolon", "buy milk; pay rent", wed, cats, []want{{title: "buy milk"}, {title: "pay rent"}}},
		{"a line break is not a separator", "buy milk\npay rent", wed, cats, []want{{title: "buy milk pay rent"}}},
		{"salt and pepper stays whole", "buy salt and pepper", wed, cats, []want{{title: "buy salt and pepper"}}},
		{"a bare and stays whole", "buy milk, and eggs", wed, cats, []want{{title: "buy milk, and eggs"}}},
		{"a trailing separator", "买牛奶；", wed, cats, []want{{title: "买牛奶"}}},
		{"each part has its own words", "提醒我周一9点体检，还有周五前交房租", wed, cats, []want{
			{"体检", "", "2026-10-26 09:00", "", false},
			{"交房租", "2026-10-23 18:00", "", "", false},
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := flat(Parse(tc.in, tc.now, tc.cats))
			if len(got) != len(tc.want) {
				t.Fatalf("Parse(%q) = %+v, want %+v", tc.in, got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("Parse(%q)[%d] = %+v, want %+v", tc.in, i, got[i], tc.want[i])
				}
			}
		})
	}
}

// The default hours are the ones the rest of the app uses.
func TestDefaultHours(t *testing.T) {
	if DueHour != 18 || RemindHour != 9 {
		t.Fatalf("DueHour, RemindHour = %d, %d, want 18, 9", DueHour, RemindHour)
	}
}
