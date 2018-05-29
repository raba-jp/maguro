package build

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/nlopes/slack"
	"github.com/vivitInc/maguro/drone"
)

const (
	ActionRepoSelect   = "build_action_repo_select"
	ActionNumberSelect = "build_action_build_number_select"
	ActionRestart      = "build_action_build_restart"
	ActionStop         = "build_action_build_stop"
	ActionCancel       = "build_action_cancel"
)

type Params struct {
	Slack   *slack.Client
	Drone   *drone.Drone
	Event   *slack.MessageEvent
	Message *slack.AttachmentActionCallback
	Action  *slack.AttachmentAction
}

func Handle(p Params) *slack.Message {
	action := *p.Action
	switch action.Name {
	case ActionRepoSelect:
		return selectBuildNumber(&p)
	case ActionNumberSelect:
		return selectAction(&p)
	case ActionRestart:
		return restart(&p)
	case ActionStop:
		return stop(&p)
	default:
		originalMessage := p.Message.OriginalMessage
		originalMessage.Attachments[0].Text = "エラーが発生したよ！"
		originalMessage.Attachments[0].Actions = []slack.AttachmentAction{}
		return &originalMessage
	}
}

func SelectRepo(p *Params) {
	repos, err := p.Drone.GetRepositories()
	if err != nil {
		log.Printf("failed to get repositories: %s", err)
	}
	options := make([]slack.AttachmentActionOption, len(repos))
	for i, repo := range repos {
		options[i] = slack.AttachmentActionOption{
			Text:  repo.FullName(),
			Value: repo.FullName(),
		}
	}

	params := slack.PostMessageParameters{
		Attachments: []slack.Attachment{
			slack.Attachment{
				Text:       "どのリポジトリにする？",
				CallbackID: "build",
				Actions: []slack.AttachmentAction{
					{
						Name:    ActionRepoSelect,
						Type:    "select",
						Options: options,
					},
					{
						Name:  ActionCancel,
						Text:  "キャンセル",
						Type:  "button",
						Style: "danger",
					},
				},
			},
		},
	}

	if _, _, err := p.Slack.PostMessage(p.Event.Channel, "", params); err != nil {
		log.Printf("failed to post message: %s", err)
	}
}

func selectBuildNumber(p *Params) *slack.Message {
	// Format: {owner}/{repo}
	value := p.Action.SelectedOptions[0].Value

	repo := drone.GetRepoFromFullName(value)
	builds := p.Drone.GetRunningBuildNumber(repo)

	options := make([]slack.AttachmentActionOption, len(builds))
	for i, build := range builds {
		options[i] = slack.AttachmentActionOption{
			Text:  fmt.Sprintf("%d: %s %s", build.Number, build.Commit, build.Message),
			Value: fmt.Sprintf("%s:%d", repo.FullName(), build.Number),
		}
	}

	originalMessage := p.Message.OriginalMessage
	originalMessage.Attachments[0].Text = fmt.Sprintf("%sのどのビルド？", value)
	originalMessage.Attachments[0].Actions = []slack.AttachmentAction{
		{
			Name:    ActionNumberSelect,
			Type:    "select",
			Options: options,
		},
		{
			Name:  ActionCancel,
			Text:  "Cancel",
			Type:  "button",
			Style: "danger",
		},
	}

	return &originalMessage
}

func selectAction(p *Params) *slack.Message {
	// Format: {owner}/{repo}:{build}
	value := p.Action.SelectedOptions[0].Value

	originalMessage := p.Message.OriginalMessage
	originalMessage.Attachments[0].Text = fmt.Sprintf("%sをどうする？", value)
	originalMessage.Attachments[0].Actions = []slack.AttachmentAction{
		{
			Name:  ActionRestart,
			Text:  "Restart",
			Type:  "button",
			Value: value,
			Style: "primary",
		},
		{
			Name:  ActionStop,
			Text:  "Stop",
			Type:  "button",
			Value: value,
			Style: "primary",
		},
		{
			Name:  ActionCancel,
			Text:  "Cancel",
			Type:  "button",
			Style: "danger",
		},
	}

	return &originalMessage
}

func restart(p *Params) *slack.Message {
	strs := strings.Split(p.Action.Value, ":")
	repo := drone.GetRepoFromFullName(strs[0])

	var title = ""
	if number, err := strconv.Atoi(strs[1]); err != nil {
		title = "再実行失敗した..."
	} else {
		if droneErr := p.Drone.RestartBuild(*repo, number); droneErr != nil {
			title = fmt.Sprintf("%dを再実行できなかった...", number)
		} else {
			title = fmt.Sprintf("%dを再実行したよ！", number)
		}
	}
	originalMessage := p.Message.OriginalMessage
	originalMessage.Attachments[0].Actions = []slack.AttachmentAction{}
	originalMessage.Attachments[0].Fields = []slack.AttachmentField{
		{
			Title: title,
			Value: "",
			Short: false,
		},
	}

	return &originalMessage
}

func stop(p *Params) *slack.Message {
	strs := strings.Split(p.Action.Value, ":")
	repo := drone.GetRepoFromFullName(strs[0])

	var title = ""
	if number, err := strconv.Atoi(strs[1]); err != nil {
		title = "止めるの失敗した..."
	} else {
		if droneErr := p.Drone.KillBuild(*repo, number); droneErr != nil {
			title = fmt.Sprintf("%dを止めるの失敗した...", number)
		} else {
			title = fmt.Sprintf("%dを止めたよ！", number)
		}
	}
	originalMessage := p.Message.OriginalMessage
	originalMessage.Attachments[0].Actions = []slack.AttachmentAction{}
	originalMessage.Attachments[0].Fields = []slack.AttachmentField{
		{
			Title: title,
			Value: "",
			Short: false,
		},
	}

	return &originalMessage
}
