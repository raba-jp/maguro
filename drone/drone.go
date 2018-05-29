package drone

import (
	"github.com/drone/drone-go/drone"
	"golang.org/x/oauth2"
)

type Drone struct {
	client drone.Client
	owner  string
}

func NewDrone(host, token, owner string) *Drone {
	config := new(oauth2.Config)
	auther := config.Client(
		oauth2.NoContext,
		&oauth2.Token{
			AccessToken: token,
		},
	)
	client := drone.NewClient(host, auther)
	return &Drone{client, owner}
}

func (d *Drone) GetRepositories() ([]Repo, error) {
	repos, err := d.client.RepoList()
	if err != nil {
		return []Repo{}, err
	}
	list := []Repo{}
	for _, r := range repos {
		list = append(list, Repo{r.Owner, r.Name})
	}
	return list, nil
}

func (d *Drone) GetRunningBuildNumber(repo *Repo) []*Build {
	builds, err := d.client.BuildList(repo.Owner, repo.Name)
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

func (d *Drone) RestartBuild(repo Repo, number int) error {
	err := d.client.BuildKill(repo.Owner, repo.Name, number)
	if err != nil {
		return err
	}
	_, res := d.client.BuildStart(repo.Owner, repo.Name, number, nil)
	return res
}

func (d *Drone) KillBuild(repo Repo, number int) error {
	return d.client.BuildKill(repo.Owner, repo.Name, number)
}

func (d *Drone) GetSucceededBuild(repo *Repo) []*Build {
	list, err := d.client.BuildList(repo.Owner, repo.Name)
	if err != nil {
		return make([]*Build, 0)
	}

	builds := []*Build{}
	for _, b := range list {
		if b.Status == "success" {
			builds = append(builds, &Build{
				b.Number,
				string([]rune(b.Commit)[:6]),
				b.Message,
			})
		}
	}
	return builds
}

func (d *Drone) Deploy(repo Repo, number int, env string, params map[string]string) (*drone.Build, error) {
	build, err := d.client.Deploy(repo.Owner, repo.Name, number, env, params)
	return build, err
}
