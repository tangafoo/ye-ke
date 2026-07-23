package anthropic

import (
	"context"
	_ "embed"
	"fmt"
	"strings"

	"byakugan/internal/store"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

type Client struct {
	api anthropic.Client
}

func New(apiKey string) *Client {
	return &Client{api: anthropic.NewClient(option.WithAPIKey(apiKey))}
}

//go:embed prompts/framing_v1.txt
var systemPrompt string

// sectionLabel is the identity line above each verbatim block in the prompt:
// "[DDA 1952 s12(2)-(4) — Restriction on import and export ...]".
func sectionLabel(h store.Hit) string {
	return fmt.Sprintf("[%s s%s — %s]", h.StatuteAbbr, h.DisplaySection(), h.Heading)
}

func generateUserContent(question string, related, hits []store.Hit) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Question: %s\n\nRetrieved sections:\n", question)
	for _, h := range hits {
		fmt.Fprintf(&sb, "%s\n%s\n\n", sectionLabel(h), h.Text)
	}

	if len(related) > 0 {
		sb.WriteString("Related sections (statutory cross-references of the sections above — use only where they actually bear on the question):\n")
		for _, h := range related {
			fmt.Fprintf(&sb, "%s\n%s\n\n", sectionLabel(h), h.Text)
		}
	}

	return sb.String()
}

// Frame asks the model to frame the retrieved law. hits are the similarity
// matches; related are statutory cross-references of those hits.
func (c *Client) Frame(ctx context.Context, question string, hits, related []store.Hit) (string, bool, error) {

	userContent := generateUserContent(question, related, hits)

	msg, err := c.api.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeSonnet4_5,
		MaxTokens: 1024,
		System:    []anthropic.TextBlockParam{{Text: systemPrompt}},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(userContent)),
		},
	})
	if err != nil {
		return "", false, fmt.Errorf("framing failed: %w", err)
	}
	truncated := msg.StopReason != anthropic.StopReasonEndTurn

	for _, block := range msg.Content {
		if t, ok := block.AsAny().(anthropic.TextBlock); ok {
			return t.Text, truncated, nil
		}
	}

	return "", false, fmt.Errorf("no text block in res")
}

func (c *Client) FrameStream(ctx context.Context, question string, hits, related []store.Hit, onDelta func(text string) error) (interrupted bool, err error) {

	userContent := generateUserContent(question, related, hits)

	stream := c.api.Messages.NewStreaming(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeSonnet4_5,
		MaxTokens: 1024,
		System:    []anthropic.TextBlockParam{{Text: systemPrompt}},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(userContent)),
		},
	})

	message := anthropic.Message{}

	for stream.Next() {
		event := stream.Current()

		if err := message.Accumulate(event); err != nil {
			fmt.Println("anthropic message could not be accumulated.")
		}

		switch ev := event.AsAny().(type) {
		case anthropic.ContentBlockDeltaEvent:
			switch d := ev.Delta.AsAny().(type) {
			case anthropic.TextDelta:
				if err := onDelta(d.Text); err != nil {
					return false, err
				}
			}
		}
	}
	if err := stream.Err(); err != nil {
		return false, fmt.Errorf("stream failed: %w", err)
	}

	switch message.StopReason {
	case anthropic.StopReasonMaxTokens:
		return true, nil
	case anthropic.StopReasonRefusal:
		return false, fmt.Errorf("anthropic refused to answer")
	}

	return false, nil
}
