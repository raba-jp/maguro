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
		originalMessage.Attachments[0].Text = fmt.Sprintf("%sをどうする？", strings.Title(value))
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
		if number, err := strconv.Atoi(action.Value); err != nil {
			responseMessage(w, message.OriginalMessage, "再実行できなかった...", "")
		} else {
			if droneErr := h.drone.RestartBuild(number); droneErr != nil {
				responseMessage(w, message.OriginalMessage, fmt.Sprintf("%dを再実行できなかった...", number), "")
			} else {
				responseMessage(w, message.OriginalMessage, fmt.Sprintf("%dを再実行したよ！", number), "")
			}
		}
	case actionKill:
		if number, err := strconv.Atoi(action.Value); err != nil {
			responseMessage(w, message.OriginalMessage, "止めるの失敗した...", "")
		} else {
			if droneErr := h.drone.KillBuild(number); droneErr != nil {
				responseMessage(w, message.OriginalMessage, fmt.Sprintf("%dを止めるの失敗した...", number), "")
			} else {
				responseMessage(w, message.OriginalMessage, fmt.Sprintf("%dを止めたよ！", number), "")
			}
		}
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
