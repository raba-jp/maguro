package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/nlopes/slack"
	"github.com/vivitInc/maguro/drone"
	"go.uber.org/zap"
)

type Build struct {
	slack *slack.Client
	drone *drone.Drone
}

func (b *Build) SelectRepo(event *slack.MessageEvent) {
	repos, err := b.drone.GetRepositories()
	if err != nil {
		logger.Error("Failed to get repositories", zap.String("detail", err.Error()))
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
					SelectMenu(BuildActionSelectRepo, options),
					CancelButton(),
				},
			},
		},
	}

	if _, _, err := b.slack.PostMessage(event.Channel, "", params); err != nil {
		logger.Error("Failed to post message", zap.String("detail", err.Error()))
	}
}

func (b *Build) SelectBuild(message *slack.AttachmentActionCallback) *slack.Message {
	originalMessage := message.OriginalMessage

	// Format: {owner}/{repo}
	value := message.Actions[0].SelectedOptions[0].Value

	repo := drone.GetRepoFromFullName(value)
	builds, err := b.drone.GetRunningBuildNumber(repo)
	if err != nil {
		logger.Error("Failed to get running build", zap.String("detail", err.Error()))
		originalMessage.Attachments = Message(fmt.Sprintf("エラーが発生したよ！\n%s", err))
		return &originalMessage
	}

	options := make([]slack.AttachmentActionOption, len(builds))
	for i, build := range builds {
		options[i] = slack.AttachmentActionOption{
			Text:  fmt.Sprintf("%d: %s %s", build.Number, build.Commit, build.Message),
			Value: fmt.Sprintf("%s:%d", repo.FullName(), build.Number),
		}
	}

	originalMessage.Attachments[0].Text = fmt.Sprintf("%sのどのビルド？", value)
	originalMessage.Attachments[0].Actions = []slack.AttachmentAction{
		SelectMenu(BuildActionSelectBuild, options),
		CancelButton(),
	}
	return &originalMessage
}

func (b *Build) SelectAction(message *slack.AttachmentActionCallback) *slack.Message {
	// Format: {owner}/{repo}:{build}
	value := message.Actions[0].SelectedOptions[0].Value

	originalMessage := message.OriginalMessage
	originalMessage.Attachments[0].Text = fmt.Sprintf("%sをどうする？", value)
	originalMessage.Attachments[0].Actions = []slack.AttachmentAction{
		PrimaryButton(BuildActionRestart, "再実行", value),
		PrimaryButton(BuildActionStop, "停止", value),
		CancelButton(),
	}
	return &originalMessage
}

func (b *Build) Restart(message *slack.AttachmentActionCallback) *slack.Message {
	originalMessage := message.OriginalMessage

	strs := strings.Split(message.Actions[0].Value, ":")
	repo := drone.GetRepoFromFullName(strs[0])
	number, err := strconv.Atoi(strs[1])
	if err != nil {
		originalMessage.Attachments = Message(fmt.Sprintf("%dを再実行できなかった...", number))
		return &originalMessage
	}
	if err := b.drone.RestartBuild(*repo, number); err != nil {
		originalMessage.Attachments = Message(fmt.Sprintf("%dを再実行できなかった...", number))
		return &originalMessage
	}

	originalMessage.Attachments = Message(fmt.Sprintf("%dを再実行したよ！", number))
	return &originalMessage
}

func (b *Build) Stop(message *slack.AttachmentActionCallback) *slack.Message {
	originalMessage := message.OriginalMessage
	strs := strings.Split(message.Actions[0].Value, ":")
	repo := drone.GetRepoFromFullName(strs[0])

	build, err := strconv.Atoi(strs[1])
	if err != nil {
		originalMessage.Attachments = Message("止めるの失敗した...")
		return &originalMessage
	}
	if err := b.drone.KillBuild(*repo, build); err != nil {
		originalMessage.Attachments = Message(fmt.Sprintf("%dを止めるの失敗した...", build))
		return &originalMessage
	}

	originalMessage.Attachments = Message(fmt.Sprintf("%dを止めたよ！", build))
	return &originalMessage
}
