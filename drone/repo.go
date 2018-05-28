package drone

import (
	"fmt"
	"strings"
)

type Repo struct {
	Owner string
	Name  string
}

func GetRepoFromFullName(name string) *Repo {
	strs := strings.Split(name, "/")
	return &Repo{strs[0], strs[1]}
}
func (r *Repo) FullName() string {
	return fmt.Sprintf("%s/%s", r.Owner, r.Name)
}
