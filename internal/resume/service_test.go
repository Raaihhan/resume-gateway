package resume

import (
	"context"
	"errors"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	resumev1 "github.com/Raaihhan/resume-gateway/gen/go/proto/resume/v1"
	"google.golang.org/protobuf/types/known/emptypb"
)

// spyRepo: implementasi Repo untuk UT
type spyRepo struct {
	// GetProfile
	getProfileResp Profile
	getProfileErr  error

	// ListProjects
	lpQ        string
	lpPage     int
	lpPageSize int
	lpItems    []Project
	lpTotal    int
	lpErr      error

	// InsertMessage
	imName  string
	imEmail string
	imMsg   string
	imErr   error

	// GetHobbies
	hTags string
	hRmk  string
	hDesc string
	hErr  error
}

func (s *spyRepo) GetProfile(ctx context.Context) (Profile, error) {
	return s.getProfileResp, s.getProfileErr
}
func (s *spyRepo) ListProjects(ctx context.Context, q string, page, pageSize int) ([]Project, int, error) {
	s.lpQ, s.lpPage, s.lpPageSize = q, page, pageSize
	return s.lpItems, s.lpTotal, s.lpErr
}
func (s *spyRepo) InsertMessage(ctx context.Context, name, email, msg string) error {
	s.imName, s.imEmail, s.imMsg = name, email, msg
	return s.imErr
}
func (s *spyRepo) GetHobbies(ctx context.Context) (tagsCSV, remarksCSV, descriptionCSV string, err error) {
	return s.hTags, s.hRmk, s.hDesc, s.hErr
}

func TestService_WithGoConvey(t *testing.T) {
	Convey("NewService mengembalikan instance", t, func() {
		s := NewService(&spyRepo{})
		So(s, ShouldNotBeNil)
	})

	Convey("GetProfile - sukses", t, func() {
		spy := &spyRepo{
			getProfileResp: Profile{
				FullName:  "Jane Doe",
				Headline:  "Backend Engineer",
				Summary:   "Loves Go",
				Location:  "Bandung",
				Skills:    []string{"Go", "MySQL"},
				AvatarURL: "https://example/avatar.png",
			},
		}
		svc := NewService(spy)

		out, err := svc.GetProfile(context.Background(), &resumev1.GetProfileRequest{})
		So(err, ShouldBeNil)
		So(out.GetFullName(), ShouldEqual, "Jane Doe")
		So(out.GetHeadline(), ShouldEqual, "Backend Engineer")
		So(out.GetSummary(), ShouldEqual, "Loves Go")
		So(out.GetLocation(), ShouldEqual, "Bandung")
		So(out.GetAvatarUrl(), ShouldEqual, "https://example/avatar.png")
		So(out.GetSkills(), ShouldResemble, []string{"Go", "MySQL"})
	})

	Convey("GetProfile - error dari repo dibubble", t, func() {
		svc := NewService(&spyRepo{getProfileErr: errors.New("db down")})
		out, err := svc.GetProfile(context.Background(), &resumev1.GetProfileRequest{})
		So(err, ShouldNotBeNil)
		So(out, ShouldBeNil)
	})

	Convey("ListProjects - sukses, parsing & clamp", t, func() {
		spy := &spyRepo{
			lpItems: []Project{
				{ID: "1", Name: "Proj A", Description: "A", RepoURL: "r1", DemoURL: "d1", Tags: []string{"a", "b"}},
				{ID: "2", Name: "Proj B", Description: "B", RepoURL: "r2", DemoURL: "d2", Tags: []string{"x"}},
			},
			lpTotal: 12,
		}
		svc := NewService(spy)
		req := &resumev1.ListProjectsRequest{
			PageNumber:  " 2 ",
			PageSize:    " 200 ", // >100 -> tetap default 10
			SearchQuery: "  go  ",
		}

		res, err := svc.ListProjects(context.Background(), req)
		So(err, ShouldBeNil)

		// argumen ke repo
		So(spy.lpQ, ShouldEqual, "go")
		So(spy.lpPage, ShouldEqual, 2)
		So(spy.lpPageSize, ShouldEqual, 10)

		// mapping result
		So(res.GetTotal(), ShouldEqual, "12")
		So(len(res.GetItems()), ShouldEqual, 2)
		So(res.GetItems()[0].GetId(), ShouldEqual, "1")
		So(res.GetItems()[0].GetName(), ShouldEqual, "Proj A")
		So(res.GetItems()[0].GetTags(), ShouldResemble, []string{"a", "b"})
		So(res.GetItems()[1].GetId(), ShouldEqual, "2")
		So(res.GetItems()[1].GetName(), ShouldEqual, "Proj B")
	})

	Convey("ListProjects - sukses, pageSize valid", t, func() {
		spy := &spyRepo{
			lpItems: []Project{{ID: "99", Name: "Only One"}},
			lpTotal: 1,
		}
		svc := NewService(spy)
		req := &resumev1.ListProjectsRequest{
			PageNumber:  "1",
			PageSize:    "7",
			SearchQuery: "",
		}

		res, err := svc.ListProjects(context.Background(), req)
		So(err, ShouldBeNil)
		So(spy.lpPage, ShouldEqual, 1)
		So(spy.lpPageSize, ShouldEqual, 7)
		So(res.GetTotal(), ShouldEqual, "1")
		So(len(res.GetItems()), ShouldEqual, 1)
		So(res.GetItems()[0].GetId(), ShouldEqual, "99")
	})

	Convey("ListProjects - error dari repo dibubble (cek default page/size/query)", t, func() {
		spy := &spyRepo{lpErr: errors.New("query failed")}
		svc := NewService(spy)
		req := &resumev1.ListProjectsRequest{
			PageNumber:  "not-a-number",
			PageSize:    "-5",
			SearchQuery: "   ",
		}

		out, err := svc.ListProjects(context.Background(), req)
		So(err, ShouldNotBeNil)
		So(out, ShouldBeNil)
		So(spy.lpPage, ShouldEqual, 1)     // default
		So(spy.lpPageSize, ShouldEqual, 10) // default
		So(spy.lpQ, ShouldEqual, "")       // trimmed kosong
	})

	Convey("CreateContactMessage - sukses", t, func() {
		spy := &spyRepo{}
		svc := NewService(spy)

		req := &resumev1.CreateContactMessageRequest{
			Name:    "Jane",
			Email:   "jane@mail.com",
			Message: "Hello",
		}
		res, err := svc.CreateContactMessage(context.Background(), req)
		So(err, ShouldBeNil)
		So(res.GetStatus(), ShouldEqual, "Send It Succesfully")
		So(spy.imName, ShouldEqual, "Jane")
		So(spy.imEmail, ShouldEqual, "jane@mail.com")
		So(spy.imMsg, ShouldEqual, "Hello")
	})

	Convey("CreateContactMessage - error dibubble", t, func() {
		spy := &spyRepo{imErr: errors.New("insert failed")}
		svc := NewService(spy)

		out, err := svc.CreateContactMessage(context.Background(), &resumev1.CreateContactMessageRequest{
			Name: "Foo", Email: "foo@mail.com", Message: "Bar",
		})
		So(err, ShouldNotBeNil)
		So(out, ShouldBeNil)
	})

	Convey("ListHobbies - sukses", t, func() {
		spy := &spyRepo{hTags: "tag1, tag2", hRmk: "r1 | r2", hDesc: "d1 || d2"}
		svc := NewService(spy)

		res, err := svc.ListHobbies(context.Background(), &emptypb.Empty{})
		So(err, ShouldBeNil)
		So(res.GetTags(), ShouldEqual, "tag1, tag2")
		So(res.GetRemaks(), ShouldEqual, "r1 | r2")
		So(res.GetDescription(), ShouldEqual, "d1 || d2")
	})

	Convey("ListHobbies - error dibubble", t, func() {
		spy := &spyRepo{hErr: errors.New("boom")}
		svc := NewService(spy)

		out, err := svc.ListHobbies(context.Background(), &emptypb.Empty{})
		So(err, ShouldNotBeNil)
		So(out, ShouldBeNil)
	})
}
