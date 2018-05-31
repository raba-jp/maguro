package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/nlopes/slack"
	"github.com/vivitInc/maguro/config"
	"github.com/vivitInc/maguro/drone"
	"go.uber.org/zap"
)

type Deploy struct {
	slack  *slack.Client
	drone  *drone.Drone
	config *config.Config
}

func DeployAttachmentFields(name, env string, build string) []slack.AttachmentField {
	return []slack.AttachmentField{
		slack.AttachmentField{
			Title: "リポジトリ",
			Value: name,
			Short: false,
		},
		slack.AttachmentField{
			Title: "環境",
			Value: env,
			Short: false,
		},
		slack.AttachmentField{
			Title: "デプロイ対象",
			Value: build,
			Short: false,
		},
	}
}

func (d *Deploy) SelectRepo(event *slack.MessageEvent) {
	repos := d.config.Repositories
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
				CallbackID: "deploy",
				Fields:     DeployAttachmentFields("", "", ""),
				Actions: []slack.AttachmentAction{
					SelectMenu(DeployActionSelectRepo, options),
					CancelButton(),
				},
			},
		},
	}

	if _, _, err := d.slack.PostMessage(event.Channel, "", params); err != nil {
		logger.Error("Failed to post message", zap.String("detail", err.Error()))
	}
}

func (d *Deploy) SelectEnv(message *slack.AttachmentActionCallback) *slack.Message {
	originalMessage := message.OriginalMessage

	// Format: {owner}/{repo}
	value := message.Actions[0].SelectedOptions[0].Value
	var repo *config.Repository
	repo = nil
	for _, r := range d.config.Repositories {
		if r.Name == value {
			repo = &r
			break
		}
	}
	if repo == nil {
		logger.Error("Failed to get environment")
		originalMessage.Attachments = Message("デプロイできる環境が見つからないよ！")
		return &originalMessage
	}

	options := make([]slack.AttachmentActionOption, len(repo.Env))
	for i, env := range repo.Env {
		options[i] = slack.AttachmentActionOption{
			Text:  fmt.Sprintf("%s", env),
			Value: fmt.Sprintf("%s:%s", value, env),
		}
	}

	originalMessage.Attachments[0].Text = fmt.Sprintf("%sのどの環境？", value)
	originalMessage.Attachments[0].Fields = DeployAttachmentFields(value, "", "")
	originalMessage.Attachments[0].Actions = []slack.AttachmentAction{
		SelectMenu(DeployActionSelectEnv, options),
		CancelButton(),
	}
	return &originalMessage
}

func (d *Deploy) SelectBuild(message *slack.AttachmentActionCallback) *slack.Message {
	originalMessage := message.OriginalMessage

	// Format: {owner}/{repo}:{env}
	strs := strings.Split(message.Actions[0].SelectedOptions[0].Value, ":")
	repo := drone.GetRepoFromFullName(strs[0])
	builds, err := d.drone.GetSucceededBuilds(repo)
	if err != nil {
		logger.Error("Failed to get succeeded builds", zap.String("detail", err.Error()))
		originalMessage.Attachments = Message("デプロイできる環境が見つからないよ！")
		return &originalMessage
	}

	options := make([]slack.AttachmentActionOption, len(builds))
	for i, build := range builds {
		options[i] = slack.AttachmentActionOption{
			Text:  fmt.Sprintf("%d: %s %s", build.Number, build.Commit, build.Message),
			Value: fmt.Sprintf("%s:%s:%d", repo.FullName(), strs[1], build.Number),
		}
	}

	originalMessage.Attachments[0].Text = fmt.Sprintf("%sのどのビルド？", repo.FullName())
	originalMessage.Attachments[0].Fields = DeployAttachmentFields(repo.FullName(), strs[1], "")
	originalMessage.Attachments[0].Actions = []slack.AttachmentAction{
		SelectMenu(DeployActionSelectBuild, options),
		CancelButton(),
	}
	return &originalMessage
}

func (d *Deploy) Confirm(message *slack.AttachmentActionCallback) *slack.Message {
	// Format: {owner}/{owner}:{env}:{number}
	value := message.Actions[0].SelectedOptions[0].Value
	strs := strings.Split(value, ":")

	originalMessage := message.OriginalMessage
	originalMessage.Attachments[0].Text = fmt.Sprintf("これをデプロイしていい？\n%s:%s -> %s", strs[0], strs[2], strs[1])
	originalMessage.Attachments[0].Fields = DeployAttachmentFields(strs[0], strs[1], strs[2])
	originalMessage.Attachments[0].Actions = []slack.AttachmentAction{
		PrimaryButton(DeployActionConfirm, "デプロイ", value),
		CancelButton(),
	}
	return &originalMessage
}

func (d *Deploy) Deploy(message *slack.AttachmentActionCallback) *slack.Message {
	originalMessage := message.OriginalMessage
	originalMessage.Attachments = Message(fmt.Sprintf("デプロイに失敗したみたい..."))

	// Format: {owner}/{owner}:{env}:{number}
	value := message.Actions[0].Value
	strs := strings.Split(value, ":")

	repo := drone.GetRepoFromFullName(strs[0])
	number, err := strconv.Atoi(strs[2])
	if err != nil {
		logger.Error("Failed to deploy", zap.String("detail", err.Error()))
		return &originalMessage
	}

	build, err := d.drone.Deploy(*repo, number, strs[1], map[string]string{})
	if err != nil {
		logger.Error("Failed to deploy", zap.String("detail", err.Error()))
		return &originalMessage
	}

	uri := fmt.Sprintf("https://ci.dev.hinata.me/%s/%d", strs[0], build.Number)
	originalMessage.Attachments[0].Text = fmt.Sprintf(`デプロイ始めたよ！
	デプロイ状況はここから見てね。
	 -> %s
	`, uri)
	originalMessage.Attachments[0].Fields = DeployAttachmentFields(strs[0], strs[1], strs[2])
	return &originalMessage
}
