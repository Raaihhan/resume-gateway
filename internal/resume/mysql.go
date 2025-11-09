package resume

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	resumev1 "github.com/Raaihhan/resume-gateway/gen/go/proto/resume/v1"
	_ "github.com/go-sql-driver/mysql"
	"github.com/opentracing/opentracing-go"
)

// Pastikan interface Repo sudah ada di package ini:
// type Repo interface {
//   GetProfile(ctx context.Context) (Profile, error)
//   ListProjects(ctx context.Context, q string, page, pageSize int) ([]Project, int, error)
//   InsertMessage(ctx context.Context, name, email, msg string) error
// }

type MySQLRepo struct {
	db *sql.DB
}

func NewMySQLRepo(dsn string) (*MySQLRepo, error) {
	span, _ := opentracing.StartSpanFromContext(context.Background(), "NewMySQLRepo")
	defer span.Finish()
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)
	span.LogKV("Ping", db)
	return &MySQLRepo{db: db}, nil
}

func (r *MySQLRepo) Close() error { return r.db.Close() }

func (r *MySQLRepo) GetProfile(ctx context.Context) (Profile, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "repo.GetProfile")
	defer span.Finish()
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	row := r.db.QueryRowContext(ctx, `
		SELECT full_name, headline, summary, location, skills_csv, avatar_url
		FROM profiles
		ORDER BY updated_at DESC
		LIMIT 1
	`)
	var p Profile
	var skillsCSV string
	if err := row.Scan(&p.FullName, &p.Headline, &p.Summary, &p.Location, &skillsCSV, &p.AvatarURL); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Profile{}, nil
		}
		return Profile{}, err
	}
	p.Skills = csvToSlice(skillsCSV)
	res := resumev1.Profile{
		FullName:  p.FullName,
		Headline:  p.Headline,
		Summary:   p.Summary,
		Location:  p.Location,
		Skills:    p.Skills,
		AvatarUrl: p.AvatarURL,
	}
	span.LogKV("Succes Get Profile", res)
	return p, nil
}

func (r *MySQLRepo) ListProjects(ctx context.Context, q string, page, pageSize int) ([]Project, int, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "repo.ListProjects")
	defer span.Finish()

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	// total
	var total int
	if q == "" {
		if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM projects`).Scan(&total); err != nil {
			return nil, 0, err
		}
	} else {
		like := "%" + q + "%"
		if err := r.db.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM projects
			WHERE name LIKE ? OR description LIKE ?
		`, like, like).Scan(&total); err != nil {
			span.LogKV("Error", err.Error())
			return nil, 0, err
		}
	}

	// items
	var (
		rows *sql.Rows
		err  error
	)
	if q == "" {
		rows, err = r.db.QueryContext(ctx, `
			SELECT id,name,description,repo_url,demo_url,tags_csv
			FROM projects
			ORDER BY created_at DESC
			LIMIT ? OFFSET ?`, pageSize, offset)
	} else {
		like := "%" + q + "%"
		rows, err = r.db.QueryContext(ctx, `
			SELECT id,name,description,repo_url,demo_url,tags_csv
			FROM projects
			WHERE name LIKE ? OR description LIKE ?
			ORDER BY created_at DESC
			LIMIT ? OFFSET ?`, like, like, pageSize, offset)
	}
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []Project
	for rows.Next() {
		var p Project
		var tagsCSV string
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.RepoURL, &p.DemoURL, &tagsCSV); err != nil {
			return nil, 0, err
		}
		p.Tags = csvToSlice(tagsCSV)
		items = append(items, p)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	span.LogKV("result.count", len(items), "total", total, "page", page, "page_size", pageSize)
	return items, total, nil
}

func (r *MySQLRepo) InsertMessage(ctx context.Context, name, email, msg string) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "repo.InsertMessage")
	defer span.Finish()
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	insert, err := r.db.ExecContext(ctx, `
		INSERT INTO contact_messages (name,email,message) VALUES (?,?,?)
	`, name, email, msg)
	span.LogKV("Successfully Insert", insert)
	return err
}

func (r *MySQLRepo) GetHobbies(ctx context.Context) (tagsCSV, remarksCSV, descriptionCSV string, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "repo.GetHobbies")
	defer span.Finish()

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	q := `
SELECT
  COALESCE(GROUP_CONCAT(IFNULL(tag, '')         ORDER BY tag SEPARATOR ', '), '')   AS tags_csv,
  COALESCE(GROUP_CONCAT(IFNULL(remarks, '')     ORDER BY tag SEPARATOR ' | '), '')  AS remarks_csv,
  COALESCE(GROUP_CONCAT(IFNULL(description, '') ORDER BY tag SEPARATOR ' || '), '') AS description_csv
FROM hobbies;
`
	span.LogKV("db.query", "GetHobbies", "sql", q)

	var t, rmk, desc sql.NullString
	if err := r.db.QueryRowContext(ctx, q).Scan(&t, &rmk, &desc); err != nil {
		span.SetTag("error", true)
		span.LogKV("error.msg", err.Error())
		return "", "", "", err
	}

	if t.Valid {
		tagsCSV = t.String
	}
	if rmk.Valid {
		remarksCSV = rmk.String
	}
	if desc.Valid {
		descriptionCSV = desc.String
	}

	span.LogKV(
		"result.tags.len", tagsCSV,
		"result.remarks.len", remarksCSV,
		"result.description.len", descriptionCSV,
		"status", "success",
	)

	return tagsCSV, remarksCSV, descriptionCSV, nil
}

func csvToSlice(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		v := strings.TrimSpace(p)
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}
