package main

import (
	"log"
	"net/http"
	"os"

	"github.com/kelseyhightower/envconfig"
	"github.com/nlopes/slack"
)

type envConfig struct {
	Port              string `envconfig:"PORT" default:"3000"`
	BotToken          string `envconfig:"BOT_TOKEN" required:"true"`
	VerificationToken string `envconfig:"VERIFICATION_TOKEN" required:"true"`
	BotID             string `envconfig:"BOT_ID" required:"true"`
	ChannelID         string `envconfig:"CHANNEL_ID" required:"true"`
	DroneToken        string `envconfig:"DRONE_TOKEN" required:"true"`
	DroneHost         string `envconfig:"DRONE_HOST" required:"true"`
	RepositoryOwner   string `envconfig:"REPOSITORY_OWNER" required:"true"`
	RepositoryName    string `envconfig:"REPOSITORY_NAME" required:"true"`
}

func main() {
	os.Exit(_main(os.Args[1:]))
}

func _main(args []string) int {
	var env envConfig
	if err := envconfig.Process("", &env); err != nil {
		log.Printf("[ERROR] Failed to process env var: %s", err)
		return 1
	}
	drone := NewDrone(
		env.DroneHost,
		env.DroneToken,
		env.RepositoryOwner,
		env.RepositoryName,
	)

	log.Printf("[INFO] Start slack event listening")
	client := slack.New(env.BotToken)
	slackListener := &SlackListener{
		client:    client,
		botID:     env.BotID,
		channelID: env.ChannelID,
		drone:     drone,
	}
	go slackListener.ListenAndResponse()

	http.Handle("/interaction", interactionHandler{
		verificationToken: env.VerificationToken,
		drone:             drone,
	})

	log.Printf("[INFO] Server listening on :%s", env.Port)
	if err := http.ListenAndServe(":"+env.Port, nil); err != nil {
		log.Printf("[ERROR] %s", err)
		return 1
	}

	return 0
}
