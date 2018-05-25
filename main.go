package main

import (
	"log"
	"net/http"
	"os"

	"github.com/kelseyhightower/envconfig"
	"github.com/nlopes/slack"
	"github.com/vivitInc/maguro/drone"
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
	d := drone.NewDrone(
		env.DroneHost,
		env.DroneToken,
		env.RepositoryOwner,
	)

	log.Printf("[INFO] Start slack event listening")
	client := slack.New(env.BotToken)
	slackListener := &SlackListener{
		client:    client,
		botID:     env.BotID,
		channelID: env.ChannelID,
		drone:     d,
	}
	go slackListener.ListenAndResponse()

	http.Handle("/maguro/interaction", interactionHandler{
		verificationToken: env.VerificationToken,
		drone:             d,
	})
	http.HandleFunc("/maguro/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{\"status\": \"OK\"}"))
	})

	// Slack slash commmands
	http.Handle("/maguro/public/", http.StripPrefix("/maguro/public/", http.FileServer(http.Dir("./public"))))
	http.HandleFunc("/maguro/toyama", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{\"attachments\": [{\"title\": \"toyama\", \"image_url\": \"https://bot.dev.hinata.me/maguro/public/toyama.jpg\"}], \"response_type\": \"in_channel\"}"))
	})
	http.HandleFunc("/maguro/tomoka", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{\"attachments\": [{\"title\": \"tomoka\", \"image_url\": \"https://bot.dev.hinata.me/maguro/public/tomoka.png\"}], \"response_type\": \"in_channel\"}"))
	})
	http.HandleFunc("/maguro/loading", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{\"attachments\": [{\"title\": \"loading\", \"image_url\": \"https://bot.dev.hinata.me/maguro/public/loading.jpg\"}], \"response_type\": \"in_channel\"}"))
	})

	log.Printf("[INFO] Server listening on :%s", env.Port)
	if err := http.ListenAndServe(":"+env.Port, nil); err != nil {
		log.Printf("[ERROR] %s", err)
		return 1
	}

	return 0
}
