package main

import (
	"github.com/drone/drone-go/drone"
	"golang.org/x/oauth2"
)

type Drone struct {
	client drone.Client
	owner  string
	name   string
}

type Build struct {
	Number  int
	Commit  string
	Message string
}

func NewDrone(host, token, owner, name string) *Drone {
	config := new(oauth2.Config)
	auther := config.Client(
		oauth2.NoContext,
		&oauth2.Token{
			AccessToken: token,
		},
	)
	client := drone.NewClient(host, auther)
	return &Drone{client, owner, name}
}

func (d *Drone) GetRunningBuildNumber() []*Build {
	builds, err := d.client.BuildList(d.owner, d.name)
	if err != nil {
		return make([]*Build, 0)
	}

	numbers := []*Build{}
	for _, b := range builds {
		if b.Status == "running" {
			numbers = append(numbers, &Build{
				b.Number,
				string([]rune(b.Commit)[:6]),
				b.Message,
			})
		}
	}

	return numbers
}

func (d *Drone) RestartBuild(number int) error {
	err := d.client.BuildKill(d.owner, d.name, number)
	if err != nil {
		return err
	}
	_, res := d.client.BuildStart(d.owner, d.name, number, nil)
	return res
}

func (d *Drone) KillBuild(number int) error {
	return d.client.BuildKill(d.owner, d.name, number)
}
