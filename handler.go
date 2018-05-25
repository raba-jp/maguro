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
	"github.com/vivitInc/maguro/drone"
)

// interactionHandler handles interactive message response.
type interactionHandler struct {
	slackClient       *slack.Client
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
	case actionRepoSelect:
		m := h.handleRepoSelect(message, &action)
		w.Header().Add("Content-type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(&m)
		return
	case actionBuildSelect:
		m := h.handleBuildSelect(message, &action)
		w.Header().Add("Content-type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(&m)
		return
	case actionBuildRestart:
		strs := strings.Split(action.Value, ":")
		repo := drone.GetRepoFromFullName(strs[0])

		if number, err := strconv.Atoi(strs[1]); err != nil {
			responseMessage(w, message.OriginalMessage, "止めるの失敗した...", "")
		} else {
			if droneErr := h.drone.RestartBuild(*repo, number); droneErr != nil {
				responseMessage(w, message.OriginalMessage, fmt.Sprintf("%dを再実行できなかった...", number), "")
			} else {
				responseMessage(w, message.OriginalMessage, fmt.Sprintf("%dを再実行したよ！", number), "")
			}
		}
	case actionBuildKill:
		strs := strings.Split(action.Value, ":")
		repo := drone.GetRepoFromFullName(strs[0])

		if number, err := strconv.Atoi(strs[1]); err != nil {
			responseMessage(w, message.OriginalMessage, "止めるの失敗した...", "")
		} else {
			if droneErr := h.drone.KillBuild(*repo, number); droneErr != nil {
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

func (h *interactionHandler) handleRepoSelect(message *slack.AttachmentActionCallback, action *slack.AttachmentAction) *slack.Message {
	// {owner}/{repo}
	value := action.SelectedOptions[0].Value

	repo := drone.GetRepoFromFullName(value)
	log.Printf("owner: %s", repo.Owner)
	log.Printf("name: %s", repo.Name)
	builds := h.drone.GetRunningBuildNumber(repo)

	options := []slack.AttachmentActionOption{}
	for _, build := range builds {
		options = append(options, slack.AttachmentActionOption{
			Text:  fmt.Sprintf("%d: %s %s", build.Number, build.Commit, build.Message),
			Value: fmt.Sprintf("%s:%d", repo.FullName(), build.Number),
		})
	}
	log.Printf("%s", options)

	originalMessage := message.OriginalMessage
	originalMessage.Attachments[0].Text = fmt.Sprintf("%sのどのビルド？", value)
	originalMessage.Attachments[0].Actions = []slack.AttachmentAction{
		{
			Name:    actionBuildSelect,
			Type:    "select",
			Options: options,
		},
		{
			Name:  actionCancel,
			Text:  "Cancel",
			Type:  "button",
			Style: "danger",
		},
	}

	return &originalMessage
}

func (h *interactionHandler) handleBuildSelect(message *slack.AttachmentActionCallback, action *slack.AttachmentAction) *slack.Message {
	// {owner}/{repo}:{build}
	value := action.SelectedOptions[0].Value

	originalMessage := message.OriginalMessage
	originalMessage.Attachments[0].Text = fmt.Sprintf("%sをどうする？", value)
	originalMessage.Attachments[0].Actions = []slack.AttachmentAction{
		{
			Name:  actionBuildRestart,
			Text:  "Restart",
			Type:  "button",
			Value: value,
			Style: "primary",
		},
		{
			Name:  actionBuildKill,
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

	return &originalMessage
}
