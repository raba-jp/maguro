package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/kelseyhightower/envconfig"
	"github.com/nlopes/slack"
	"github.com/robfig/cron"
	"github.com/vivitInc/maguro/config"
	"github.com/vivitInc/maguro/drone"
	"go.uber.org/zap"
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

var logger *zap.Logger

func main() {
	os.Exit(_main(os.Args[1:]))
}

func _main(args []string) int {
	if err := initLogger(); err != nil {
		fmt.Printf("Failed to initialize logger %s", err)
		return 1
	}

	env, err := initEnvConfig()
	if err != nil {
		logger.Error("Failed to process env var", zap.String("detail", err.Error()))
		return 1
	}

	conf, err := config.LoadConfig()
	if err != nil {
		logger.Error("Failed to load config", zap.String("detail", err.Error()))
		return 1
	}

	d := drone.NewDrone(
		env.DroneHost,
		env.DroneToken,
		env.RepositoryOwner,
	)
	client := slack.New(env.BotToken)
	slackListener := &SlackListener{
		client:    client,
		botID:     env.BotID,
		channelID: env.ChannelID,
		drone:     d,
		config:    conf,
	}

	http.Handle("/maguro/interaction", interactionHandler{
		slack:             client,
		verificationToken: env.VerificationToken,
		drone:             d,
		config:            conf,
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
	http.HandleFunc("/maguro/loading", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{\"attachments\": [{\"title\": \"loading\", \"image_url\": \"https://bot.dev.hinata.me/maguro/public/loading.jpg\"}], \"response_type\": \"in_channel\"}"))
	})

	logger.Info("Start scheduler")
	InitScheduler(d, &conf.Schedules)

	logger.Info("Start slack event listening")
	go slackListener.ListenAndResponse()

	logger.Info("Server listening", zap.String("port", env.Port))
	if err := http.ListenAndServe(":"+env.Port, nil); err != nil {
		logger.Error("Any error raised", zap.String("detail", err.Error()))
		return 1
	}

	return 0
}

func initLogger() error {
	var err error
	logger, err = zap.NewProduction()
	if err != nil {
		return err
	}
	defer logger.Sync()
	return nil
}

func initEnvConfig() (*envConfig, error) {
	var env envConfig
	if err := envconfig.Process("", &env); err != nil {
		return nil, err
	}
	return &env, nil
}

func InitScheduler(client *drone.Drone, schedules *[]config.Schedule) {
	cron := cron.New()
	for _, s := range *schedules {
		logger.Info("Register function", zap.String("repo", s.Name), zap.String("cron", s.Cron))
		repo := drone.GetRepoFromFullName(s.Name)
		cron.AddFunc(s.Cron, func() {
			client.RestartSucceededMasterBuild(*repo)
		})
	}
	cron.Start()
}
