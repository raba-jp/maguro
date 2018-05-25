package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/nlopes/slack"
)

const (
	// action is used for slack attament action.
	actionSelect  = "select"
	actionRestart = "restart"
	actionKill    = "kill"
	actionCancel  = "cancel"
)

type SlackListener struct {
	client    *slack.Client
	botID     string
	channelID string
	drone     *Drone
}

func (s *SlackListener) ListenAndResponse() {
	rtm := s.client.NewRTM()

	// Start listening slack events
	go rtm.ManageConnection()

	// Handle slack events
	for msg := range rtm.IncomingEvents {
		switch ev := msg.Data.(type) {
		case *slack.MessageEvent:
			if err := s.handleMessageEvent(ev); err != nil {
				log.Printf("[ERROR] Failed to handle message: %s", err)
			}
		}
	}
}

func (s *SlackListener) handleMessageEvent(ev *slack.MessageEvent) error {
	// Only response in specific channel. Ignore else.
	if ev.Channel != s.channelID {
		log.Printf("%s %s", ev.Channel, ev.Msg.Text)
		return nil
	}

	// Only response mention to bot. Ignore else.
	if !strings.HasPrefix(ev.Msg.Text, fmt.Sprintf("<@%s> ", s.botID)) {
		return nil
	}

	// Parse message
	m := strings.Split(strings.TrimSpace(ev.Msg.Text), " ")[1:]
	if len(m) == 0 || m[0] != "build" {
		return fmt.Errorf("invalid message")
	}

	builds := s.drone.GetRunningBuildNumber()
	options := []slack.AttachmentActionOption{}
	for _, build := range builds {
		options = append(options, slack.AttachmentActionOption{
			Text:  fmt.Sprintf("%d: %s %s", build.Number, build.Commit, build.Message),
			Value: fmt.Sprintf("%d", build.Number),
		})
	}

	// value is passed to message handler when request is approved.
	attachment := slack.Attachment{
		Text:       "どのビルド？",
		Color:      "#f9a41b",
		CallbackID: "build",
		Actions: []slack.AttachmentAction{
			{
				Name:    actionSelect,
				Type:    "select",
				Options: options,
			},

			{
				Name:  actionCancel,
				Text:  "Cancel",
				Type:  "button",
				Style: "danger",
			},
		},
	}

	params := slack.PostMessageParameters{
		Attachments: []slack.Attachment{
			attachment,
		},
	}

	if _, _, err := s.client.PostMessage(ev.Channel, "", params); err != nil {
		return fmt.Errorf("failed to post message: %s", err)
	}

	return nil
}
