package agent

import (
	"context"
	"testing"
	"time"

	"github.com/cloudwego/eino/schema"
)

func TestAskSendsPicturesWithTheText(t *testing.T) {
	r := newRig(t, "UTC", time.Date(2026, 10, 14, 9, 0, 0, 0, time.UTC), say("Seen."))
	if _, err := r.agent.Ask(context.Background(), "what is in these?", Scope{},
		Image{MIME: "image/png", Data: []byte("one")}, Image{MIME: "image/jpeg", Data: []byte("two")}); err != nil {
		t.Fatal(err)
	}
	in := r.model.sent(0)
	user := in[len(in)-1]
	if user.Role != schema.User || user.Content != "" || len(user.UserInputMultiContent) != 3 {
		t.Fatalf("user message = %+v", user)
	}
	if p := user.UserInputMultiContent[0]; p.Type != schema.ChatMessagePartTypeText || p.Text != "what is in these?" {
		t.Errorf("first part = %+v", p)
	}
	for i, want := range []struct{ mime, data string }{{"image/png", "b25l"}, {"image/jpeg", "dHdv"}} {
		p := user.UserInputMultiContent[i+1]
		if p.Type != schema.ChatMessagePartTypeImageURL || p.Image == nil || p.Image.MIMEType != want.mime || *p.Image.Base64Data != want.data {
			t.Errorf("image part %d = %+v", i, p)
		}
	}
}

func TestAskWithOnlyPicturesHasNoTextPartAndWithoutPicturesStaysPlain(t *testing.T) {
	r := newRig(t, "UTC", time.Date(2026, 10, 14, 9, 0, 0, 0, time.UTC), say("Seen."), say("Hi."))
	if _, err := r.agent.Ask(context.Background(), "  ", Scope{}, Image{MIME: "image/png", Data: []byte("x")}); err != nil {
		t.Fatal(err)
	}
	if u := r.model.sent(0); len(u[len(u)-1].UserInputMultiContent) != 1 {
		t.Errorf("picture-only message = %+v", u[len(u)-1])
	}
	r.ask("plain")
	if u := r.model.sent(1); u[len(u)-1].Content != "plain" || len(u[len(u)-1].UserInputMultiContent) != 0 {
		t.Errorf("text-only message = %+v", u[len(u)-1])
	}
}
