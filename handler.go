package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/nlopes/slack"
	"github.com/vivitInc/maguro/config"
	"github.com/vivitInc/maguro/drone"
	"go.uber.org/zap"
)

// interactionHandler handles interactive message response.
type interactionHandler struct {
	slack             *slack.Client
	verificationToken string
	drone             *drone.Drone
	config            *config.Config
}

func (h interactionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	message, errCode := h.validate(r)
	if errCode != 0 {
		w.WriteHeader(errCode)
		return
	}

	action := message.Actions[0]
	build := Build{slack: h.slack, drone: h.drone}
	deploy := Deploy{slack: h.slack, drone: h.drone, config: h.config}
	switch action.Name {
	case BuildActionSelectRepo:
		responseMessage(w, build.SelectBuild(message))
	case BuildActionSelectBuild:
		responseMessage(w, build.SelectAction(message))
	case BuildActionRestart:
		responseMessage(w, build.Restart(message))
	case BuildActionStop:
		responseMessage(w, build.Stop(message))
	case DeployActionSelectRepo:
		responseMessage(w, deploy.SelectEnv(message))
	case DeployActionSelectEnv:
		responseMessage(w, deploy.SelectBuild(message))
	case DeployActionSelectBuild:
		responseMessage(w, deploy.Confirm(message))
	case DeployActionConfirm:
		responseMessage(w, deploy.Deploy(message))
	case ActionCancel:
		originalMessage := message.OriginalMessage
		originalMessage.Attachments = Message("やっぱりやめた！", "")
		responseMessage(w, &originalMessage)
	default:
		logger.Error("Invalid action", zap.String("action", action.Name))
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (h *interactionHandler) validate(r *http.Request) (*slack.AttachmentActionCallback, int) {
	if r.Method != http.MethodPost {
		logger.Error("Invalid method", zap.String("name", r.Method))
		return nil, http.StatusMethodNotAllowed
	}

	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logger.Error("Failed to read request body", zap.String("detail", err.Error()))
		return nil, http.StatusInternalServerError
	}

	jsonStr, err := url.QueryUnescape(string(buf)[8:])
	if err != nil {
		logger.Error("[ERROR] Failed to unespace request body", zap.String("detail", err.Error()))
		return nil, http.StatusInternalServerError
	}

	var message slack.AttachmentActionCallback
	if err := json.Unmarshal([]byte(jsonStr), &message); err != nil {
		logger.Error("Failed to decode json message from slack", zap.String("detail", jsonStr))
		return nil, http.StatusInternalServerError
	}

	// Only accept message from slack with valid token
	if message.Token != h.verificationToken {
		logger.Error("Invalid token", zap.String("token", message.Token))
		return nil, http.StatusUnauthorized
	}

	return &message, 0
}

func responseMessage(w http.ResponseWriter, original *slack.Message) {
	w.Header().Add("Content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(&original)
}
