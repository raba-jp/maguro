package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/nlopes/slack"
	"github.com/vivitInc/maguro/drone"
)

const (
	// action is used for slack attament action.
	actionRepoSelect   = "repo_select"
	actionBuildSelect  = "build_select"
	actionBuildRestart = "build_restart"
	actionBuildKill    = "build_kill"
	actionCancel       = "cancel"
)

type SlackListener struct {
	client    *slack.Client
	botID     string
	channelID string
	drone     *drone.Drone
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
	if len(m) == 0 {
		return fmt.Errorf("invalid message")
	}

	switch m[0] {
	case "build":
		if err := s.handleBuild(ev); err != nil {
			log.Printf("failed to handle build: %s", err)
		}
	case "tomoka", "ともか":
		if err := s.handleTomoka(ev); err != nil {
			log.Printf("failed to handle tomoka: %s", err)
		}
	}

	return nil
}

func (s *SlackListener) handleBuild(ev *slack.MessageEvent) error {
	repos, err := s.drone.GetRepositories()
	if err != nil {
		return fmt.Errorf("failed to get repositories: %s", err)
	}

	options := []slack.AttachmentActionOption{}
	for _, repo := range repos {
		options = append(options, slack.AttachmentActionOption{
			Text:  repo.FullName(),
			Value: repo.FullName(),
		})
	}
	attachment := slack.Attachment{
		Text:       "どのリポジトリにする？",
		CallbackID: "build",
		Actions: []slack.AttachmentAction{
			{
				Name:    actionRepoSelect,
				Type:    "select",
				Options: options,
			},
			{
				Name:  actionCancel,
				Text:  "キャンセル",
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

func (s *SlackListener) handleTomoka(ev *slack.MessageEvent) error {
	params := slack.PostMessageParameters{
		Attachments: []slack.Attachment{
			{
				Title:    "tomoka",
				ImageURL: "https://bot.dev.hinata.me/maguro/public/tomoka.png",
			},
		},
	}
	if _, _, err := s.client.PostMessage(ev.Channel, "", params); err != nil {
		return fmt.Errorf("failed to post message: %s", err)
	}
	return nil
}
