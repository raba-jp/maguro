package main

import (
	"github.com/nlopes/slack"
	"go.uber.org/zap"
)

func Tomoka(client *slack.Client, ev *slack.MessageEvent) {
	params := slack.PostMessageParameters{
		Attachments: []slack.Attachment{
			{
				Title:    "tomoka",
				ImageURL: "https://bot.dev.hinata.me/maguro/public/tomoka.png",
			},
		},
	}
	if _, _, err := client.PostMessage(ev.Channel, "", params); err != nil {
		logger.Error("Failed to post message", zap.String("detail", err.Error()))
	}
}
