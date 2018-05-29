package main

import (
	"fmt"
	"strings"

	"github.com/nlopes/slack"
	"github.com/vivitInc/maguro/config"
	"github.com/vivitInc/maguro/drone"
	"go.uber.org/zap"
)

type SlackListener struct {
	client    *slack.Client
	botID     string
	channelID string
	drone     *drone.Drone
	config    *config.Config
}

func (s *SlackListener) ListenAndResponse() {
	rtm := s.client.NewRTM()

	// Start listening slack events
	go rtm.ManageConnection()

	// Handle slack events
	for msg := range rtm.IncomingEvents {
		switch ev := msg.Data.(type) {
		case *slack.MessageEvent:
			s.handleMessageEvent(ev)
		}
	}
}

func (s *SlackListener) handleMessageEvent(ev *slack.MessageEvent) {
	// Only response mention to bot. Ignore else.
	if !strings.HasPrefix(ev.Msg.Text, fmt.Sprintf("<@%s> ", s.botID)) {
		return
	}

	var allowed = false
	for _, c := range s.config.Channels {
		if c == ev.Channel {
			allowed = true
			break
		}
	}
	if !allowed {
		logger.Info(
			"Channel ID don't match",
			zap.String("channel", ev.Channel),
			zap.String("text", ev.Msg.Text),
		)
		return
	}

	// Parse message
	m := strings.Split(strings.TrimSpace(ev.Msg.Text), " ")[1:]
	if len(m) == 0 {
		logger.Error("Invalid message", zap.String("detail", ev.Msg.Text))
		return
	}

	switch m[0] {
	case "build":
		b := Build{slack: s.client, drone: s.drone}
		b.SelectRepo(ev)
		return
	case "tomoka", "ともか":
		Tomoka(s.client, ev)
		return
	case "deploy":
		d := Deploy{slack: s.client, drone: s.drone, config: s.config}
		d.SelectRepo(ev)
		return
	}
}
