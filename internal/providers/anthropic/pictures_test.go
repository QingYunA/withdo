package anthropic_test

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/schema"
)

func TestUserPicturesBecomeImageBlocksAndAPictureOnlyMessageIsKept(t *testing.T) {
	a := newAPI(t, 200, textReply)
	m := a.model(t)
	b64 := "aGVsbG8="
	pic := func(mime string) schema.MessageInputPart {
		return schema.MessageInputPart{Type: schema.ChatMessagePartTypeImageURL, Image: &schema.MessageInputImage{
			MessagePartCommon: schema.MessagePartCommon{Base64Data: &b64, MIMEType: mime}}}
	}
	history := []*schema.Message{
		{Role: schema.User, UserInputMultiContent: []schema.MessageInputPart{
			{Type: schema.ChatMessagePartTypeText, Text: "move these to Friday"}, pic("image/png"), pic("image/jpeg")}},
		{Role: schema.Assistant, Content: "Done."},
		{Role: schema.User, UserInputMultiContent: []schema.MessageInputPart{pic("image/png")}},
	}
	if _, err := m.Generate(context.Background(), history); err != nil {
		t.Fatal(err)
	}
	msgs := list(a.body["messages"])
	if len(msgs) != 3 {
		t.Fatalf("a message with only a picture must be kept: %v", msgs)
	}
	first := list(obj(msgs[0])["content"])
	if len(first) != 3 || obj(first[0])["type"] != "text" || obj(first[0])["text"] != "move these to Friday" {
		t.Fatalf("first message = %v", msgs[0])
	}
	for i, mime := range []string{"image/png", "image/jpeg"} {
		b := obj(first[i+1])
		src := obj(b["source"])
		if b["type"] != "image" || src["type"] != "base64" || src["media_type"] != mime || src["data"] != b64 {
			t.Errorf("image block %d = %v", i, b)
		}
	}
	last := list(obj(msgs[2])["content"])
	if len(last) != 1 || obj(last[0])["type"] != "image" {
		t.Errorf("picture-only message = %v", msgs[2])
	}
}
