package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"

	"github.com/nlopes/slack"
	"github.com/vivitInc/maguro/build"
	"github.com/vivitInc/maguro/drone"
)

// interactionHandler handles interactive message response.
type interactionHandler struct {
	slack             *slack.Client
	verificationToken string
	drone             *drone.Drone
}

func (h interactionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	message, errCode := h.validate(r)
	if errCode != 0 {
		w.WriteHeader(errCode)
		return
	}

	action := message.Actions[0]
	switch action.Name {
	case build.ActionRepoSelect, build.ActionNumberSelect, build.ActionSelect, build.ActionRestart, build.ActionStop:
		params := build.Params{
			Slack:   h.slack,
			Drone:   h.drone,
			Message: message,
			Action:  &action,
		}
		message := build.Handle(action.Name, params)

		w.Header().Add("Content-type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(&message)
		return
	case build.ActionCancel:
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

func (h *interactionHandler) validate(r *http.Request) (*slack.AttachmentActionCallback, int) {
	if r.Method != http.MethodPost {
		log.Printf("[ERROR] Invalid method: %s", r.Method)
		return nil, http.StatusMethodNotAllowed
	}

	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("[ERROR] Failed to read request body: %s", err)
		return nil, http.StatusInternalServerError
	}

	jsonStr, err := url.QueryUnescape(string(buf)[8:])
	if err != nil {
		log.Printf("[ERROR] Failed to unespace request body: %s", err)
		return nil, http.StatusInternalServerError
	}

	var message slack.AttachmentActionCallback
	if err := json.Unmarshal([]byte(jsonStr), &message); err != nil {
		log.Printf("[ERROR] Failed to decode json message from slack: %s", jsonStr)
		return nil, http.StatusInternalServerError
	}

	// Only accept message from slack with valid token
	if message.Token != h.verificationToken {
		log.Printf("[ERROR] Invalid token: %s", message.Token)
		return nil, http.StatusUnauthorized
	}

	return &message, 0
}
