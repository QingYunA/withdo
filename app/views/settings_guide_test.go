package views

import (
	"strings"
	"testing"
)

func TestAnotherAgentRowCopiesTheSetupGuide(t *testing.T) {
	for _, tc := range []struct {
		locale, label, line, button, copied string
	}{
		{"en", "Another agent", "Not in the list? Copy this guide and paste it to the agent: it adds Withdo itself.", "Copy setup guide", "Copied"},
		{"zh-Hans", "其他 Agent", "不在列表里？复制这份说明发给它，它会自己接上 Withdo。", "复制接入说明", "已拷贝"},
	} {
		t.Run(tc.locale, func(t *testing.T) {
			r := newSettingsRig(t)
			r.pane(t, "Agents")
			r.tt.SetLocale(tc.locale)
			r.tt.Frame()
			r.tt.Scroll(340, 300, 0, 3000)
			r.tt.Frame()
			has(t, r.tt, tc.label, tc.line, tc.button)
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
	mustNil(t, r.tt.Click("Install the MCP in Zed"))
	r.rec.drain(t)
	r.tt.Frame()
	r.tt.Scroll(340, 300, 0, 330)
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
