package resume

import (
	"context"
	"database/sql"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	. "github.com/smartystreets/goconvey/convey"
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
	Convey("GetProfile - sukses", t, func() {
		repo, mock, cleanup := newMockRepo(t)
		defer cleanup()

		query := `(?s)SELECT\s+full_name,\s*headline,\s*summary,\s*location,\s*skills_csv,\s*avatar_url\s+FROM\s+profiles\s+ORDER\s+BY\s+updated_at\s+DESC\s+LIMIT\s+1`

		rows := sqlmock.NewRows([]string{
			"full_name", "headline", "summary", "location", "skills_csv", "avatar_url",
		}).AddRow("John Doe", "Software Engineer", "Summary here", "Jakarta", "Go, Docker , K8s", "https://cdn/avatar.png")

		mock.ExpectQuery(query).WillReturnRows(rows)

		got, err := repo.GetProfile(context.Background())
		So(err, ShouldBeNil)

		So(got.FullName, ShouldEqual, "John Doe")
		So(got.Headline, ShouldEqual, "Software Engineer")
		So(got.Summary, ShouldEqual, "Summary here")
		So(got.Location, ShouldEqual, "Jakarta")
		So(got.AvatarURL, ShouldEqual, "https://cdn/avatar.png")
		So(got.Skills, ShouldResemble, []string{"Go", "Docker", "K8s"})

		So(mock.ExpectationsWereMet(), ShouldBeNil)
	})
}

func TestGetProfile_NoRows(t *testing.T) {
	Convey("GetProfile - tanpa baris (return zero Profile)", t, func() {
		repo, mock, cleanup := newMockRepo(t)
		defer cleanup()

		query := `(?s)SELECT\s+full_name,\s*headline,\s*summary,\s*location,\s*skills_csv,\s*avatar_url\s+FROM\s+profiles\s+ORDER\s+BY\s+updated_at\s+DESC\s+LIMIT\s+1`

		rows := sqlmock.NewRows([]string{
			"full_name", "headline", "summary", "location", "skills_csv", "avatar_url",
		})
		mock.ExpectQuery(query).WillReturnRows(rows)

		got, err := repo.GetProfile(context.Background())
		So(err, ShouldBeNil)
		So(got, ShouldResemble, Profile{})

		So(mock.ExpectationsWereMet(), ShouldBeNil)
	})
}

func TestListProjects_NoQuery(t *testing.T) {
	Convey("ListProjects - tanpa query", t, func() {
		repo, mock, cleanup := newMockRepo(t)
		defer cleanup()

		countQuery := `(?s)SELECT\s+COUNT\(\*\)\s+FROM\s+projects`
		mock.ExpectQuery(countQuery).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

		itemsQuery := `(?s)SELECT\s+id,\s*name,\s*description,\s*repo_url,\s*demo_url,\s*tags_csv\s+FROM\s+projects\s+ORDER\s+BY\s+created_at\s+DESC\s+LIMIT\s+\?\s+OFFSET\s+\?`

		rows := sqlmock.NewRows([]string{"id", "name", "description", "repo_url", "demo_url", "tags_csv"}).
			AddRow(int64(1), "Project A", "Desc A", "https://repoA", "https://demoA", "tag1, tag2").
			AddRow(int64(2), "Project B", "Desc B", "https://repoB", "https://demoB", "tag3")

		mock.ExpectQuery(itemsQuery).WithArgs(10, 0).WillReturnRows(rows)

		items, total, err := repo.ListProjects(context.Background(), "", 1, 10)
		So(err, ShouldBeNil)
		So(total, ShouldEqual, 2)
		So(items, ShouldHaveLength, 2)
		So(items[0].Tags, ShouldResemble, []string{"tag1", "tag2"})

		So(mock.ExpectationsWereMet(), ShouldBeNil)
	})
}

func TestListProjects_WithQuery(t *testing.T) {
	Convey("ListProjects - dengan query LIKE", t, func() {
		repo, mock, cleanup := newMockRepo(t)
		defer cleanup()

		q := "go"
		like := "%" + q + "%"
		page := 2
		pageSize := 5
		offset := (page - 1) * pageSize

		countQuery := `(?s)SELECT\s+COUNT\(\*\)\s+FROM\s+projects\s+WHERE\s+name\s+LIKE\s+\?\s+OR\s+description\s+LIKE\s+\?`
		mock.ExpectQuery(countQuery).WithArgs(like, like).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

		itemsQuery := `(?s)SELECT\s+id,\s*name,\s*description,\s*repo_url,\s*demo_url,\s*tags_csv\s+FROM\s+projects\s+WHERE\s+name\s+LIKE\s+\?\s+OR\s+description\s+LIKE\s+\?\s+ORDER\s+BY\s+created_at\s+DESC\s+LIMIT\s+\?\s+OFFSET\s+\?`

		rows := sqlmock.NewRows([]string{"id", "name", "description", "repo_url", "demo_url", "tags_csv"}).
			AddRow(int64(10), "Go Project", "Search result", "", "", "go, tool")

		mock.ExpectQuery(itemsQuery).WithArgs(like, like, pageSize, offset).WillReturnRows(rows)

		items, total, err := repo.ListProjects(context.Background(), q, page, pageSize)
		So(err, ShouldBeNil)
		So(total, ShouldEqual, 3)
		So(items, ShouldHaveLength, 1)
		So(items[0].Name, ShouldEqual, "Go Project")
		So(items[0].Tags, ShouldResemble, []string{"go", "tool"})

		So(mock.ExpectationsWereMet(), ShouldBeNil)
	})
}

func TestListProjects_PageBoundsAndClamp(t *testing.T) {
	Convey("ListProjects - clamp pageSize & perbaiki page", t, func() {
		repo, mock, cleanup := newMockRepo(t)
		defer cleanup()

		mock.ExpectQuery(`(?s)SELECT\s+COUNT\(\*\)\s+FROM\s+projects`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

		itemsQuery := `(?s)SELECT\s+id,\s*name,\s*description,\s*repo_url,\s*demo_url,\s*tags_csv\s+FROM\s+projects\s+ORDER\s+BY\s+created_at\s+DESC\s+LIMIT\s+\?\s+OFFSET\s+\?`
		mock.ExpectQuery(itemsQuery).WithArgs(100, 0).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "description", "repo_url", "demo_url", "tags_csv"}))

		_, _, err := repo.ListProjects(context.Background(), "", 0, 1000)
		So(err, ShouldBeNil)
		So(mock.ExpectationsWereMet(), ShouldBeNil)
	})
}

func TestInsertMessage_Success(t *testing.T) {
	Convey("InsertMessage - sukses", t, func() {
		repo, mock, cleanup := newMockRepo(t)
		defer cleanup()

		insertQuery := `INSERT\s+INTO\s+contact_messages\s*\(name,email,message\)\s*VALUES\s*\(\?,\?,\?\)`
		mock.ExpectExec(insertQuery).
			WithArgs("Jane", "jane@mail.com", "Hello there").
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.InsertMessage(context.Background(), "Jane", "jane@mail.com", "Hello there")
		So(err, ShouldBeNil)

		So(mock.ExpectationsWereMet(), ShouldBeNil)
	})
}

func TestInsertMessage_DBError(t *testing.T) {
	Convey("InsertMessage - error dari DB", t, func() {
		repo, mock, cleanup := newMockRepo(t)
		defer cleanup()

		insertQuery := `INSERT\s+INTO\s+contact_messages\s*\(name,email,message\)\s*VALUES\s*\(\?,\?,\?\)`
		mock.ExpectExec(insertQuery).
			WithArgs("Jane", "jane@mail.com", "Hello there").
			WillReturnError(sql.ErrConnDone)

		err := repo.InsertMessage(context.Background(), "Jane", "jane@mail.com", "Hello there")
		So(err, ShouldNotBeNil)

		So(mock.ExpectationsWereMet(), ShouldBeNil)
	})
}

func TestGetHobbies_Success(t *testing.T) {
	Convey("GetHobbies - sukses", t, func() {
		repo, mock, cleanup := newMockRepo(t)
		defer cleanup()

		query := `(?s)FROM\s+hobbies;?`
		rows := sqlmock.NewRows([]string{"tags_csv", "remarks_csv", "description_csv"}).
			AddRow("tag1, tag2", "rmk1 | rmk2", "desc1 || desc2")

		mock.ExpectQuery(query).WillReturnRows(rows)

		tags, remarks, desc, err := repo.GetHobbies(context.Background())
		So(err, ShouldBeNil)
		So(tags, ShouldEqual, "tag1, tag2")
		So(remarks, ShouldEqual, "rmk1 | rmk2")
		So(desc, ShouldEqual, "desc1 || desc2")

		So(mock.ExpectationsWereMet(), ShouldBeNil)
	})
}

func TestGetHobbies_Nulls(t *testing.T) {
	Convey("GetHobbies - NULLs jadi empty string", t, func() {
		repo, mock, cleanup := newMockRepo(t)
		defer cleanup()

		query := `(?s)FROM\s+hobbies;?`
		rows := sqlmock.NewRows([]string{"tags_csv", "remarks_csv", "description_csv"}).
			AddRow(nil, nil, nil)

		mock.ExpectQuery(query).WillReturnRows(rows)

		tags, remarks, desc, err := repo.GetHobbies(context.Background())
		So(err, ShouldBeNil)
		So(tags, ShouldEqual, "")
		So(remarks, ShouldEqual, "")
		So(desc, ShouldEqual, "")

		So(mock.ExpectationsWereMet(), ShouldBeNil)
	})
}

func Test_csvToSlice(t *testing.T) {
	Convey("csvToSlice - handle empty & trimming", t, func() {
		So(csvToSlice(""), ShouldBeNil)

		in := " a, ,b , c ,  ,"
		out := csvToSlice(in)
		So(out, ShouldResemble, []string{"a", "b", "c"})
	})
}

func TestListProjects_QueryErrorBubblesUp(t *testing.T) {
	Convey("ListProjects - error pada count query di-bubble up", t, func() {
		repo, mock, cleanup := newMockRepo(t)
		defer cleanup()

		countQuery := `(?s)SELECT\s+COUNT\(\*\)\s+FROM\s+projects\s+WHERE\s+name\s+LIKE\s+\?\s+OR\s+description\s+LIKE\s+\?`
		mock.ExpectQuery(countQuery).WithArgs("%err%", "%err%").WillReturnError(sql.ErrConnDone)

		_, _, err := repo.ListProjects(context.Background(), "err", 1, 10)
		So(err, ShouldNotBeNil)

		So(mock.ExpectationsWereMet(), ShouldBeNil)
	})
}

func TestListProjects_ItemsQueryError(t *testing.T) {
	Convey("ListProjects - error pada items query di-bubble up", t, func() {
		repo, mock, cleanup := newMockRepo(t)
		defer cleanup()

		mock.ExpectQuery(`(?s)SELECT\s+COUNT\(\*\)\s+FROM\s+projects`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		itemsQuery := `(?s)SELECT\s+id,\s*name,\s*description,\s*repo_url,\s*demo_url,\s*tags_csv\s+FROM\s+projects\s+ORDER\s+BY\s+created_at\s+DESC\s+LIMIT\s+\?\s+OFFSET\s+\?`
		mock.ExpectQuery(itemsQuery).WithArgs(10, 0).
			WillReturnError(sql.ErrTxDone)

		_, _, err := repo.ListProjects(context.Background(), "", 1, 10)
		So(err, ShouldNotBeNil)

		So(mock.ExpectationsWereMet(), ShouldBeNil)
	})
}

func TestQueryRegexesCompile(t *testing.T) {
	Convey("Regex SQL valid/compile", t, func() {
		regexes := []string{
			`(?s)SELECT\s+full_name,\s*headline,\s*summary,\s*location,\s*skills_csv,\s*avatar_url\s+FROM\s+profiles\s+ORDER\s+BY\s+updated_at\s+DESC\s+LIMIT\s+1`,
			`(?s)SELECT\s+COUNT\(\*\)\s+FROM\s+projects`,
			`(?s)SELECT\s+COUNT\(\*\)\s+FROM\s+projects\s+WHERE\s+name\s+LIKE\s+\?\s+OR\s+description\s+LIKE\s+\?`,
			`(?s)SELECT\s+id,\s*name,\s*description,\s*repo_url,\s*demo_url,\s*tags_csv\s+FROM\s+projects\s+ORDER\s+BY\s+created_at\s+DESC\s+LIMIT\s+\?\s+OFFSET\s+\?`,
			`(?s)SELECT\s+id,\s*name,\s*description,\s*repo_url,\s*demo_url,\s*tags_csv\s+FROM\s+projects\s+WHERE\s+name\s+LIKE\s+\?\s+OR\s+description\s+LIKE\s+\?\s+ORDER\s+BY\s+created_at\s+DESC\s+LIMIT\s+\?\s+OFFSET\s+\?`,
			`(?s)FROM\s+hobbies;?`,
		}
		for _, r := range regexes {
			_, err := regexp.Compile(r)
			So(err, ShouldBeNil)
		}
	})
}

func TestNewMySQLRepo_InvalidDSN(t *testing.T) {
	Convey("NewMySQLRepo - DSN invalid memunculkan error", t, func() {
		repo, err := NewMySQLRepo("invalid")
		So(err, ShouldNotBeNil)
		So(repo, ShouldBeNil)
	})
}

func TestNewMySQLRepo_SuccessAndClose(t *testing.T) {
	Convey("NewMySQLRepo - sukses dan Close()", t, func() {
		// DSN format valid; tidak ada Ping() di constructor
		dsn := "user@tcp(127.0.0.1:3306)/"

		repo, err := NewMySQLRepo(dsn)
		So(err, ShouldBeNil)
		So(repo, ShouldNotBeNil)
		So(repo.db, ShouldNotBeNil)

		So(repo.db.Stats().MaxOpenConnections, ShouldEqual, 25)

		So(repo.Close(), ShouldBeNil)
	})
}
