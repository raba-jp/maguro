package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/nlopes/slack"
	"github.com/vivitInc/maguro/build"
	"github.com/vivitInc/maguro/drone"
	"github.com/vivitInc/maguro/tomoka"
)

const (
	// action is used for slack attament action.
	actionRepoSelect       = "repo_select"
	actionBuildSelect      = "build_select"
	actionBuildRestart     = "build_restart"
	actionBuildKill        = "build_kill"
	actionDeployRepoSelect = "deploy_repo_select"
	actionCancel           = "cancel"
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
		params := build.Params{Slack: s.client, Drone: s.drone, Event: ev}
		build.SelectRepo(&params)
	case "tomoka", "ともか":
		params := tomoka.Params{Slack: s.client, Event: ev}
		if err := tomoka.Handle(&params); err != nil {
			log.Printf("failed to handle tomoka: %s", err)
			return err
		}
	}

	return nil
}
