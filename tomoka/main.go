package tomoka

import (
	"fmt"

	"github.com/nlopes/slack"
)

type Params struct {
	Slack *slack.Client
	Event *slack.MessageEvent
}

func Handle(p *Params) error {
	params := slack.PostMessageParameters{
		Attachments: []slack.Attachment{
			{
				Title:    "tomoka",
				ImageURL: "https://bot.dev.hinata.me/maguro/public/tomoka.png",
			},
		},
	}
	if _, _, err := p.Slack.PostMessage(p.Event.Channel, "", params); err != nil {
		return fmt.Errorf("failed to post message: %s", err)
	}
	return nil
}
