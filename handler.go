package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/nlopes/slack"
)

// interactionHandler handles interactive message response.
type interactionHandler struct {
	slackClient       *slack.Client
	verificationToken string
	drone             *Drone
}

func (h interactionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		log.Printf("[ERROR] Invalid method: %s", r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("[ERROR] Failed to read request body: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	jsonStr, err := url.QueryUnescape(string(buf)[8:])
	if err != nil {
		log.Printf("[ERROR] Failed to unespace request body: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var message slack.AttachmentActionCallback
	if err := json.Unmarshal([]byte(jsonStr), &message); err != nil {
		log.Printf("[ERROR] Failed to decode json message from slack: %s", jsonStr)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Only accept message from slack with valid token
	if message.Token != h.verificationToken {
		log.Printf("[ERROR] Invalid token: %s", message.Token)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	action := message.Actions[0]
	switch action.Name {
	case actionSelect:
		value := action.SelectedOptions[0].Value

		originalMessage := message.OriginalMessage
		originalMessage.Attachments[0].Text = fmt.Sprintf("%sのビルドを再実行するよ？", strings.Title(value))
		originalMessage.Attachments[0].Actions = []slack.AttachmentAction{
			{
				Name:  actionRestart,
				Text:  "Restart",
				Type:  "button",
				Value: value,
				Style: "primary",
			},
			{
				Name:  actionKill,
				Text:  "Stop",
				Type:  "button",
				Value: value,
				Style: "primary",
			},
			{
				Name:  actionCancel,
				Text:  "Cancel",
				Type:  "button",
				Style: "danger",
			},
		}

		w.Header().Add("Content-type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(&originalMessage)
		return
	case actionRestart:
		title := fmt.Sprintf("再実行したよ！")

		value := action.SelectedOptions[0].Value
		number, err := strconv.Atoi(value)
		if err != nil {
			title = "再実行できなかった..."
			responseMessage(w, message.OriginalMessage, title, "")
			return
		}
		if droneErr := h.drone.RestartBuild(number); droneErr != nil {
			title = "再実行できなかった..."
		}
		responseMessage(w, message.OriginalMessage, title, "")
		return
	case actionKill:
		title := fmt.Sprintf("止めたよ！")

		value := action.SelectedOptions[0].Value
		number, err := strconv.Atoi(value)
		if err != nil {
			title = "止めるの失敗した..."
			responseMessage(w, message.OriginalMessage, title, "")
			return
		}
		if droneErr := h.drone.KillBuild(number); droneErr != nil {
			title = "止めるの失敗した..."
		}
		responseMessage(w, message.OriginalMessage, title, "")
		return
	case actionCancel:
		title := "やっぱりやめた！"
		responseMessage(w, message.OriginalMessage, title, "")
		return
	default:
		log.Printf("[ERROR] ]Invalid action was submitted: %s", action.Name)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

// responseMessage response to the original slackbutton enabled message.
// It removes button and replace it with message which indicate how bot will work
func responseMessage(w http.ResponseWriter, original slack.Message, title, value string) {
	original.Attachments[0].Actions = []slack.AttachmentAction{} // empty buttons
	original.Attachments[0].Fields = []slack.AttachmentField{
		{
			Title: title,
			Value: value,
			Short: false,
		},
	}

	w.Header().Add("Content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(&original)
}