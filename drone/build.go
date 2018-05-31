package drone

type Build struct {
	Number  int
	Commit  string
	Message string
	Status  string
}
