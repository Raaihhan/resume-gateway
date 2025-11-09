package resume

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"reflect"

	"github.com/DATA-DOG/go-sqlmock"
)

// helper: bikin repo dengan db mock
func newMockRepo(t *testing.T) (*MySQLRepo, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	return &MySQLRepo{db: db}, mock, func() { db.Close() }
}

func TestGetProfile_Success(t *testing.T) {
	repo, mock, cleanup := newMockRepo(t)
	defer cleanup()

	query := `(?s)SELECT\s+full_name,\s*headline,\s*summary,\s*location,\s*skills_csv,\s*avatar_url\s+FROM\s+profiles\s+ORDER\s+BY\s+updated_at\s+DESC\s+LIMIT\s+1`

	rows := sqlmock.NewRows([]string{
		"full_name", "headline", "summary", "location", "skills_csv", "avatar_url",
	}).AddRow("John Doe", "Software Engineer", "Summary here", "Jakarta", "Go, Docker , K8s", "https://cdn/avatar.png")

	mock.ExpectQuery(query).WillReturnRows(rows)

	got, err := repo.GetProfile(context.Background())
	if err != nil {
		t.Fatalf("GetProfile() error = %v", err)
	}

	if got.FullName != "John Doe" ||
		got.Headline != "Software Engineer" ||
		got.Summary != "Summary here" ||
		got.Location != "Jakarta" ||
		got.AvatarURL != "https://cdn/avatar.png" {
		t.Fatalf("GetProfile() got unexpected struct: %+v", got)
	}

	wantSkills := []string{"Go", "Docker", "K8s"}
	if len(got.Skills) != len(wantSkills) {
		t.Fatalf("skills length mismatch: got %v want %v", got.Skills, wantSkills)
	}
	for i := range wantSkills {
		if got.Skills[i] != wantSkills[i] {
			t.Fatalf("skills[%d] = %q, want %q", i, got.Skills[i], wantSkills[i])
		}
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestGetProfile_NoRows(t *testing.T) {
    repo, mock, cleanup := newMockRepo(t)
    defer cleanup()

    query := `(?s)SELECT\s+full_name,\s*headline,\s*summary,\s*location,\s*skills_csv,\s*avatar_url\s+FROM\s+profiles\s+ORDER\s+BY\s+updated_at\s+DESC\s+LIMIT\s+1`

    rows := sqlmock.NewRows([]string{
        "full_name", "headline", "summary", "location", "skills_csv", "avatar_url",
    })
    mock.ExpectQuery(query).WillReturnRows(rows)

    got, err := repo.GetProfile(context.Background())
    if err != nil {
        t.Fatalf("GetProfile() error = %v", err)
    }

    // ⬇️ pake DeepEqual karena ada slice
    if !reflect.DeepEqual(got, Profile{}) {
        t.Fatalf("GetProfile() expected zero Profile, got: %+v", got)
    }

    if err := mock.ExpectationsWereMet(); err != nil {
        t.Fatalf("unmet expectations: %v", err)
    }
}

func TestListProjects_NoQuery(t *testing.T) {
	repo, mock, cleanup := newMockRepo(t)
	defer cleanup()

	// count
	countQuery := `(?s)SELECT\s+COUNT\(\*\)\s+FROM\s+projects`
	mock.
		ExpectQuery(countQuery).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	// items
	itemsQuery := `(?s)SELECT\s+id,\s*name,\s*description,\s*repo_url,\s*demo_url,\s*tags_csv\s+FROM\s+projects\s+ORDER\s+BY\s+created_at\s+DESC\s+LIMIT\s+\?\s+OFFSET\s+\?`

	rows := sqlmock.NewRows([]string{"id", "name", "description", "repo_url", "demo_url", "tags_csv"}).
		AddRow(int64(1), "Project A", "Desc A", "https://repoA", "https://demoA", "tag1, tag2").
		AddRow(int64(2), "Project B", "Desc B", "https://repoB", "https://demoB", "tag3")

	// default page=1, pageSize=10 -> limit=10 offset=0
	mock.ExpectQuery(itemsQuery).WithArgs(10, 0).WillReturnRows(rows)

	items, total, err := repo.ListProjects(context.Background(), "", 1, 10)
	if err != nil {
		t.Fatalf("ListProjects() error = %v", err)
	}
	if total != 2 {
		t.Fatalf("total = %d, want 2", total)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	if items[0].Tags == nil || len(items[0].Tags) != 2 || items[0].Tags[0] != "tag1" || items[0].Tags[1] != "tag2" {
		t.Fatalf("unexpected tags parsed: %+v", items[0].Tags)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestListProjects_WithQuery(t *testing.T) {
	repo, mock, cleanup := newMockRepo(t)
	defer cleanup()

	q := "go"
	like := "%" + q + "%"
	page := 2
	pageSize := 5
	offset := (page - 1) * pageSize

	// count with WHERE
	countQuery := `(?s)SELECT\s+COUNT\(\*\)\s+FROM\s+projects\s+WHERE\s+name\s+LIKE\s+\?\s+OR\s+description\s+LIKE\s+\?`
	mock.ExpectQuery(countQuery).WithArgs(like, like).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

	// items with WHERE + LIMIT/OFFSET
	itemsQuery := `(?s)SELECT\s+id,\s*name,\s*description,\s*repo_url,\s*demo_url,\s*tags_csv\s+FROM\s+projects\s+WHERE\s+name\s+LIKE\s+\?\s+OR\s+description\s+LIKE\s+\?\s+ORDER\s+BY\s+created_at\s+DESC\s+LIMIT\s+\?\s+OFFSET\s+\?`

	rows := sqlmock.NewRows([]string{"id", "name", "description", "repo_url", "demo_url", "tags_csv"}).
		AddRow(int64(10), "Go Project", "Search result", "", "", "go, tool")

	mock.ExpectQuery(itemsQuery).WithArgs(like, like, pageSize, offset).WillReturnRows(rows)

	items, total, err := repo.ListProjects(context.Background(), q, page, pageSize)
	if err != nil {
		t.Fatalf("ListProjects() error = %v", err)
	}
	if total != 3 {
		t.Fatalf("total = %d, want 3", total)
	}
	if len(items) != 1 || items[0].Name != "Go Project" {
		t.Fatalf("unexpected items: %+v", items)
	}
	if len(items[0].Tags) != 2 || items[0].Tags[0] != "go" || items[0].Tags[1] != "tool" {
		t.Fatalf("unexpected tags: %+v", items[0].Tags)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestListProjects_PageBoundsAndClamp(t *testing.T) {
	repo, mock, cleanup := newMockRepo(t)
	defer cleanup()

	// page <= 0 -> page=1; pageSize > 100 -> clamp to 100; q empty
	mock.ExpectQuery(`(?s)SELECT\s+COUNT\(\*\)\s+FROM\s+projects`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	itemsQuery := `(?s)SELECT\s+id,\s*name,\s*description,\s*repo_url,\s*demo_url,\s*tags_csv\s+FROM\s+projects\s+ORDER\s+BY\s+created_at\s+DESC\s+LIMIT\s+\?\s+OFFSET\s+\?`
	// expect limit=100 (clamped), offset=0 (page becomes 1)
	mock.ExpectQuery(itemsQuery).WithArgs(100, 0).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "description", "repo_url", "demo_url", "tags_csv"}))

	_, _, err := repo.ListProjects(context.Background(), "", 0, 1000)
	if err != nil {
		t.Fatalf("ListProjects() error = %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestInsertMessage_Success(t *testing.T) {
	repo, mock, cleanup := newMockRepo(t)
	defer cleanup()

	insertQuery := `INSERT\s+INTO\s+contact_messages\s*\(name,email,message\)\s*VALUES\s*\(\?,\?,\?\)`
	mock.ExpectExec(insertQuery).
		WithArgs("Jane", "jane@mail.com", "Hello there").
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := repo.InsertMessage(context.Background(), "Jane", "jane@mail.com", "Hello there"); err != nil {
		t.Fatalf("InsertMessage() error = %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestInsertMessage_DBError(t *testing.T) {
	repo, mock, cleanup := newMockRepo(t)
	defer cleanup()

	insertQuery := `INSERT\s+INTO\s+contact_messages\s*\(name,email,message\)\s*VALUES\s*\(\?,\?,\?\)`
	mock.ExpectExec(insertQuery).
		WithArgs("Jane", "jane@mail.com", "Hello there").
		WillReturnError(sql.ErrConnDone)

	err := repo.InsertMessage(context.Background(), "Jane", "jane@mail.com", "Hello there")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestGetHobbies_Success(t *testing.T) {
	repo, mock, cleanup := newMockRepo(t)
	defer cleanup()

	query := `(?s)FROM\s+hobbies;?`
	rows := sqlmock.NewRows([]string{"tags_csv", "remarks_csv", "description_csv"}).
		AddRow("tag1, tag2", "rmk1 | rmk2", "desc1 || desc2")

	mock.ExpectQuery(query).WillReturnRows(rows)

	tags, remarks, desc, err := repo.GetHobbies(context.Background())
	if err != nil {
		t.Fatalf("GetHobbies() error = %v", err)
	}
	if tags != "tag1, tag2" || remarks != "rmk1 | rmk2" || desc != "desc1 || desc2" {
		t.Fatalf("unexpected result: %q | %q | %q", tags, remarks, desc)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestGetHobbies_Nulls(t *testing.T) {
	repo, mock, cleanup := newMockRepo(t)
	defer cleanup()

	query := `(?s)FROM\s+hobbies;?`
	rows := sqlmock.NewRows([]string{"tags_csv", "remarks_csv", "description_csv"}).
		// return NULLs
		AddRow(nil, nil, nil)

	mock.ExpectQuery(query).WillReturnRows(rows)

	tags, remarks, desc, err := repo.GetHobbies(context.Background())
	if err != nil {
		t.Fatalf("GetHobbies() error = %v", err)
	}
	if tags != "" || remarks != "" || desc != "" {
		t.Fatalf("expected empty strings for NULLs, got: %q | %q | %q", tags, remarks, desc)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func Test_csvToSlice(t *testing.T) {
	// empty -> nil
	if out := csvToSlice(""); out != nil {
		t.Fatalf("csvToSlice(\"\") = %#v, want nil", out)
	}

	// trims and ignores empties
	in := " a, ,b , c ,  ,"
	out := csvToSlice(in)
	want := []string{"a", "b", "c"}
	if len(out) != len(want) {
		t.Fatalf("len = %d, want %d (%v)", len(out), len(want), out)
	}
	for i := range want {
		if out[i] != want[i] {
			t.Fatalf("out[%d]=%q, want %q", i, out[i], want[i])
		}
	}
}

func TestListProjects_QueryErrorBubblesUp(t *testing.T) {
	repo, mock, cleanup := newMockRepo(t)
	defer cleanup()

	// cause error on count query when q != ""
	countQuery := `(?s)SELECT\s+COUNT\(\*\)\s+FROM\s+projects\s+WHERE\s+name\s+LIKE\s+\?\s+OR\s+description\s+LIKE\s+\?`
	mock.ExpectQuery(countQuery).WithArgs("%err%", "%err%").WillReturnError(sql.ErrConnDone)

	_, _, err := repo.ListProjects(context.Background(), "err", 1, 10)
	if err == nil {
		t.Fatalf("expected error from count query, got nil")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestListProjects_ItemsQueryError(t *testing.T) {
	repo, mock, cleanup := newMockRepo(t)
	defer cleanup()

	// count ok
	mock.ExpectQuery(`(?s)SELECT\s+COUNT\(\*\)\s+FROM\s+projects`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	// items error
	itemsQuery := `(?s)SELECT\s+id,\s*name,\s*description,\s*repo_url,\s*demo_url,\s*tags_csv\s+FROM\s+projects\s+ORDER\s+BY\s+created_at\s+DESC\s+LIMIT\s+\?\s+OFFSET\s+\?`
	mock.ExpectQuery(itemsQuery).WithArgs(10, 0).
		WillReturnError(sql.ErrTxDone)

	_, _, err := repo.ListProjects(context.Background(), "", 1, 10)
	if err == nil {
		t.Fatalf("expected error from items query, got nil")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// Optional: verify regex compiles (sanity for maintainers tweaking queries)
func TestQueryRegexesCompile(t *testing.T) {
	regexes := []string{
		`(?s)SELECT\s+full_name,\s*headline,\s*summary,\s*location,\s*skills_csv,\s*avatar_url\s+FROM\s+profiles\s+ORDER\s+BY\s+updated_at\s+DESC\s+LIMIT\s+1`,
		`(?s)SELECT\s+COUNT\(\*\)\s+FROM\s+projects`,
		`(?s)SELECT\s+COUNT\(\*\)\s+FROM\s+projects\s+WHERE\s+name\s+LIKE\s+\?\s+OR\s+description\s+LIKE\s+\?`,
		`(?s)SELECT\s+id,\s*name,\s*description,\s*repo_url,\s*demo_url,\s*tags_csv\s+FROM\s+projects\s+ORDER\s+BY\s+created_at\s+DESC\s+LIMIT\s+\?\s+OFFSET\s+\?`,
		`(?s)SELECT\s+id,\s*name,\s*description,\s*repo_url,\s*demo_url,\s*tags_csv\s+FROM\s+projects\s+WHERE\s+name\s+LIKE\s+\?\s+OR\s+description\s+LIKE\s+\?\s+ORDER\s+BY\s+created_at\s+DESC\s+LIMIT\s+\?\s+OFFSET\s+\?`,
		`(?s)FROM\s+hobbies;?`,
	}
	for _, r := range regexes {
		if _, err := regexp.Compile(r); err != nil {
			t.Fatalf("bad regex %q: %v", r, err)
		}
	}
}
