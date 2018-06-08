package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

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

func DeployAttachmentFields(name, env string, build, target string) []slack.AttachmentField {
	return []slack.AttachmentField{
		slack.AttachmentField{
			Title: "リポジトリ",
			Value: name,
			Short: true,
		},
		slack.AttachmentField{
			Title: "環境",
			Value: env,
			Short: true,
		},
		slack.AttachmentField{
			Title: "デプロイ対象",
			Value: build,
			Short: true,
		},
		slack.AttachmentField{
			Title: "デプロイ番号",
			Value: target,
			Short: true,
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
				Fields:     DeployAttachmentFields("", "", "", ""),
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
		originalMessage.Attachments = Message("デプロイできる環境が見つからないよ！", "danger")
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
	originalMessage.Attachments[0].Fields = DeployAttachmentFields(value, "", "", "")
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
		originalMessage.Attachments = Message("デプロイできる環境が見つからないよ！", "danger")
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
	originalMessage.Attachments[0].Fields = DeployAttachmentFields(repo.FullName(), strs[1], "", "")
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
	originalMessage.Attachments[0].Text = "デプロイしていい？"
	originalMessage.Attachments[0].Fields = DeployAttachmentFields(strs[0], strs[1], strs[2], "")
	originalMessage.Attachments[0].Actions = []slack.AttachmentAction{
		PrimaryButton(DeployActionConfirm, "デプロイ", value),
		CancelButton(),
	}
	return &originalMessage
}

func (d *Deploy) Deploy(message *slack.AttachmentActionCallback) *slack.Message {
	originalMessage := message.OriginalMessage
	originalMessage.Attachments = Message(fmt.Sprintf("デプロイに失敗したみたい..."), "danger")

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
	buildNumber := strconv.Itoa(build.Number)
	if err != nil {
		logger.Error("Failed to deploy", zap.String("detail", err.Error()))
		return &originalMessage
	}

	uri := fmt.Sprintf("https://ci.dev.hinata.me/%s/%d", strs[0], build.Number)
	originalMessage.Attachments[0].Text = fmt.Sprintf(`デプロイ始めたよ！
	デプロイ状況はここから見てね。
	 -> %s
	`, uri)
	originalMessage.Attachments[0].Color = "warning"
	originalMessage.Attachments[0].Fields = DeployAttachmentFields(strs[0], strs[1], strs[2], buildNumber)

	go d.notice(*repo, strs[1], strs[2], buildNumber, message.Channel.ID, message.ResponseURL)

	return &originalMessage
}

func (d *Deploy) notice(repo drone.Repo, env, from, target, channel, url string) {
	params := slack.PostMessageParameters{
		Attachments: []slack.Attachment{
			slack.Attachment{
				Text:   "",
				Fields: DeployAttachmentFields(repo.FullName(), env, from, target),
				Color:  "good",
			},
		},
	}

	postMessage := func(params slack.PostMessageParameters) {
		input, err := json.Marshal(params)
		if err != nil {
			logger.Info("Failed to unexpected error", zap.String("detail", err.Error()))
		}

		_, err = http.Post(url, "application/json", bytes.NewBuffer(input))
		if err != nil {
			logger.Info("Failed to unexpected error", zap.String("detail", err.Error()))
		}
		if err != nil {
			logger.Error(err.Error())
		}
	}

	for {
		num, err := strconv.Atoi(target)
		if err != err {
			logger.Info("Failed to unexpected error", zap.String("detail", err.Error()))
		}

		build, err := d.drone.GetBuild(&repo, num)
		if err != nil {
			params.Attachments[0].Text = "デプロイに失敗したみたい..."
			params.Attachments[0].Color = "danger"
			postMessage(params)
			break
		}
		if build.Status == "failure" {
			params.Attachments[0].Text = "デプロイに失敗したみたい..."
			params.Attachments[0].Color = "danger"
			postMessage(params)
			break
		}
		if build.Status == "running" {
			time.Sleep(5)
			continue
		}
		if build.Status == "success" {
			postMessage(params)
			d.slack.PostMessage(channel, "", slack.PostMessageParameters{
				Attachments: []slack.Attachment{
					slack.Attachment{
						Text:  "<!here> デプロイ終わったよー",
						Color: "good",
					},
				},
			})
			break
		}
	}
}
