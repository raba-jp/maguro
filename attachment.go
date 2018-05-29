package main

import "github.com/nlopes/slack"

func SelectMenu(name string, options []slack.AttachmentActionOption) slack.AttachmentAction {
	return slack.AttachmentAction{
		Name:    name,
		Type:    "select",
		Options: options,
	}
}

func PrimaryButton(name, text, value string) slack.AttachmentAction {
	return slack.AttachmentAction{
		Name:  name,
		Text:  text,
		Value: value,
		Type:  "button",
		Style: "primary",
	}
}

func CancelButton() slack.AttachmentAction {
	return slack.AttachmentAction{
		Name:  ActionCancel,
		Text:  "キャンセル",
		Type:  "button",
		Style: "danger",
	}
}

func Message(text string) []slack.Attachment {
	return []slack.Attachment{
		slack.Attachment{
			Text:    text,
			Actions: []slack.AttachmentAction{},
		},
	}
}
