package resume

type Profile struct {
	FullName  string
	Headline  string
	Summary   string
	Location  string
	Skills    []string
	AvatarURL string
}

type Project struct {
	ID          string
	Name        string
	Description string
	RepoURL     string
	DemoURL     string
	Tags        []string
}
