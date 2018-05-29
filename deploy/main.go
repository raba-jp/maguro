package deploy

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/nlopes/slack"
	"github.com/vivitInc/maguro/config"
	"github.com/vivitInc/maguro/drone"
)

const (
	ActionRepoSelect   = "deploy_action_repo_select"
	ActionEnvSelect    = "deploy_action_env_select"
	ActionNumberSelect = "deploy_action_number_select"
	ActionConfirm      = "deploy_action_confirm"
	ActionCancel       = "cancel"
)

type Params struct {
	Slack        *slack.Client
	Drone        *drone.Drone
	Event        *slack.MessageEvent
	Message      *slack.AttachmentActionCallback
	Action       *slack.AttachmentAction
	Repositories *[]config.Repository
}

func Handle(p Params) *slack.Message {
	action := *p.Action
	switch action.Name {
	case ActionRepoSelect:
		return selectEnv(&p)
	case ActionEnvSelect:
		return selectBuildNumber(&p)
	case ActionNumberSelect:
		return confirm(&p)
	case ActionConfirm:
		return deploy(&p)
	default:
		originalMessage := p.Message.OriginalMessage
		originalMessage.Attachments[0].Text = "エラーが発生したよ！"
		originalMessage.Attachments[0].Actions = []slack.AttachmentAction{}
		return &originalMessage
	}
}

func SelectRepo(p *Params) error {
	repos := *p.Repositories

	options := make([]slack.AttachmentActionOption, len(repos))
	for i, repo := range repos {
		options[i] = slack.AttachmentActionOption{
			Text:  repo.Name,
			Value: repo.Name,
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
		return fmt.Errorf("failed to post message: %s", err)
	}
	return nil
}

func selectEnv(p *Params) *slack.Message {
	// Format: {owner}/{repo}
	value := p.Action.SelectedOptions[0].Value
	var repo *config.Repository
	repo = nil
	for _, r := range *p.Repositories {
		if r.Name == value {
			repo = &r
			break
		}
	}
	if repo == nil {
		log.Printf("failed to get env")
		originalMessage := p.Message.OriginalMessage
		originalMessage.Attachments[0].Text = "デプロイできる環境が見つからないよ！"
		originalMessage.Attachments[0].Actions = []slack.AttachmentAction{}
		return &originalMessage
	}

	options := make([]slack.AttachmentActionOption, len(repo.Env))
	for i, env := range repo.Env {
		options[i] = slack.AttachmentActionOption{
			Text:  fmt.Sprintf("%s", env),
			Value: fmt.Sprintf("%s:%s", value, env),
		}
	}

	originalMessage := p.Message.OriginalMessage
	originalMessage.Attachments[0].Text = fmt.Sprintf("%sのどの環境？", value)
	originalMessage.Attachments[0].Actions = []slack.AttachmentAction{
		{
			Name:    ActionEnvSelect,
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

func selectBuildNumber(p *Params) *slack.Message {
	// Format: {owner}/{repo}:{env}
	strs := strings.Split(p.Action.SelectedOptions[0].Value, ":")

	repo := drone.GetRepoFromFullName(strs[0])
	builds := p.Drone.GetSucceededBuild(repo)

	options := make([]slack.AttachmentActionOption, len(builds))
	for i, build := range builds {
		options[i] = slack.AttachmentActionOption{
			Text:  fmt.Sprintf("%d: %s %s", build.Number, build.Commit, build.Message),
			Value: fmt.Sprintf("%s:%s:%d", repo.FullName(), strs[1], build.Number),
		}
	}

	originalMessage := p.Message.OriginalMessage
	originalMessage.Attachments[0].Text = fmt.Sprintf("%sのどのビルド？", repo.FullName())
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

func confirm(p *Params) *slack.Message {
	// Format: {owner}/{owner}:{env}:{number}
	value := p.Action.SelectedOptions[0].Value
	strs := strings.Split(value, ":")

	originalMessage := p.Message.OriginalMessage
	originalMessage.Attachments[0].Text = fmt.Sprintf("これをデプロイしていい？\n%s:%s -> %s", strs[0], strs[2], strs[1])
	originalMessage.Attachments[0].Actions = []slack.AttachmentAction{
		{
			Name:  ActionConfirm,
			Text:  "Deploy",
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

func deploy(p *Params) *slack.Message {
	originalMessage := p.Message.OriginalMessage
	originalMessage.Attachments[0].Actions = []slack.AttachmentAction{}

	// Format: {owner}/{owner}:{env}:{number}
	value := p.Action.Value
	strs := strings.Split(value, ":")

	repo := drone.GetRepoFromFullName(strs[0])
	number, err := strconv.Atoi(strs[2])
	if err != nil {
		originalMessage.Attachments[0].Text = fmt.Sprintf("デプロイに失敗したみたい...\n%s", err)
		return &originalMessage
	}

	if n, err := p.Drone.Deploy(*repo, number, strs[1], map[string]string{}); err != nil {
		originalMessage.Attachments[0].Text = fmt.Sprintf("デプロイに失敗したみたい...\n%s", err)
	} else {
		uri := fmt.Sprintf("https://ci.dev.hinata.me/%s/%d", strs[0], n.Number)
		originalMessage.Attachments[0].Text = fmt.Sprintf("これをデプロイしてるよ！\n%s:%s -> %s\n ビルドの状況はここから見てね。\n -> %s", strs[0], strs[2], strs[1], uri)
	}
	return &originalMessage
}
