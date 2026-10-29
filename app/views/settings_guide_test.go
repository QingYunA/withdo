package views

import (
	"strings"
	"testing"
)

func TestConnectByHandCopiesTheSetupGuide(t *testing.T) {
	for _, tc := range []struct {
		locale, label, line, button, copied string
	}{
		{"en", "Connect by hand", "The command, and a guide for an agent that is not in the list", "Copy setup guide", "Copied"},
		{"zh-Hans", "手动接入", "命令行、接入说明，给列表里没有的 Agent", "复制接入说明", "已拷贝"},
	} {
		t.Run(tc.locale, func(t *testing.T) {
			r := newSettingsRig(t)
			r.tt.SetLocale(tc.locale)
			r.s.Show(PaneAgents)
			r.tt.Frame()
			has(t, r.tt, tc.label, tc.line)
			mustNil(t, r.tt.Click(tc.label))
			r.tt.Frame()
			has(t, r.tt, tc.button)
			mustNil(t, r.tt.Click(tc.button))
			if got := r.tt.Clipboard(); got != r.mac.env.Guide() || !strings.Contains(got, r.mac.env.CorePath()) {
				t.Errorf("clipboard %q", got)
			}
			r.tt.Frame()
			has(t, r.tt, tc.copied)
		})
	}
}

func TestTheManualSnippetOfAFailedInstallHasTheGuideButtonToo(t *testing.T) {
	r := newSettingsRig(t)
	r.mac.write(".config/zed/settings.json", "// my settings\n{\n  \"theme\": \"One\", // dark\n}\n")
	r.pane(t, "Agents")
	mustNil(t, r.tt.Click("Connect Zed"))
	r.rec.drain(t)
	r.tt.Frame()
	has(t, r.tt, "won't rewrite it", "Copy setup guide")
	mustNil(t, r.tt.Click("Copy command"))
	if clip := r.tt.Clipboard(); !strings.Contains(clip, `"context_servers"`) {
		t.Errorf("the snippet's Copy copied %q", clip)
	}
	r.tt.Frame()
	mustNil(t, r.tt.Click("Copy setup guide"))
	r.tt.Frame()
	if got := r.tt.Clipboard(); got != r.mac.env.Guide() {
		t.Errorf("clipboard %q", got)
	}
}
