package resume

import (
	"context"
)

type Repo interface {
	GetProfile(ctx context.Context) (Profile, error)
	ListProjects(ctx context.Context, q string, page, pageSize int) ([]Project, int, error)
	InsertMessage(ctx context.Context, name, email, msg string) error
	GetHobbies(ctx context.Context) (tagsCSV, remarksCSV, descriptionCSV string, err error)
}
