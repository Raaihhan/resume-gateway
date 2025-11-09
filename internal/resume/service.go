package resume

import (
	"context"
	"strconv"
	"strings"

	resumev1 "github.com/Raaihhan/resume-gateway/gen/go/proto/resume/v1"
	"github.com/opentracing/opentracing-go"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Service struct {
	resumev1.UnimplementedResumeGatewayServiceServer
	repo Repo
}

func NewService(r Repo) *Service { return &Service{repo: r} }

func (s *Service) GetProfile(ctx context.Context, _ *resumev1.GetProfileRequest) (*resumev1.Profile, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "Service.GetProfile")
	defer span.Finish()
	res:= &resumev1.Profile{}
	p, err := s.repo.GetProfile(ctx)
	if err != nil {
		return nil, err
	}
	res.FullName = p.FullName
	res.Headline = p.Headline
	res.Summary = p.Summary
	res.Location = p.Location
	res.Skills = p.Skills
	res.AvatarUrl = p.AvatarURL

	span.LogKV("result.profile", res)
	return res, nil

}

func (s *Service) ListProjects(ctx context.Context, req *resumev1.ListProjectsRequest) (*resumev1.ListProjectsResponse, error) {
	page := 1
	if v, err := strconv.Atoi(strings.TrimSpace(req.GetPageNumber())); err == nil && v > 0 {
		page = v
	}
	ps := 10
	if v, err := strconv.Atoi(strings.TrimSpace(req.GetPageSize())); err == nil && v > 0 && v <= 100 {
		ps = v
	}
	q := strings.TrimSpace(req.GetSearchQuery())

	items, total, err := s.repo.ListProjects(ctx, q, page, ps)
	if err != nil {
		return nil, err
	}

	out := make([]*resumev1.Project, 0, len(items))
	for _, pr := range items {
		out = append(out, &resumev1.Project{
			Id: pr.ID, Name: pr.Name, Description: pr.Description,
			RepoUrl: pr.RepoURL, DemoUrl: pr.DemoURL, Tags: pr.Tags,
		})
	}
	return &resumev1.ListProjectsResponse{Items: out, Total: strconv.Itoa(total)}, nil
}

func (s *Service) CreateContactMessage(ctx context.Context, req *resumev1.CreateContactMessageRequest) (*resumev1.CreateContactMessageResponse, error) {
	if err := s.repo.InsertMessage(ctx, req.GetName(), req.GetEmail(), req.GetMessage()); err != nil {
		return nil, err
	}
	return &resumev1.CreateContactMessageResponse{Status: "Send It Succesfully"}, nil
}

func (s *Service) ListHobbies(ctx context.Context, req *emptypb.Empty) (*resumev1.LoadHobbyResponse, error) {
	span, ctxWithSpan := opentracing.StartSpanFromContext(ctx, "Service.ListHobbies")
	defer span.Finish()

	span.LogKV("event", "load_hobbies")

	tags, remarks, description, err := s.repo.GetHobbies(ctxWithSpan)
	if err != nil {
		span.SetTag("error", true)
		span.LogKV("error.msg", err.Error())
		return nil, err
	}

	span.LogKV(
		"hobbies.tags.len", len(tags),
		"hobbies.remarks.len", len(remarks),
	)

	return &resumev1.LoadHobbyResponse{
		Tags:        tags,
		Remaks:      remarks,
		Description: description,
	}, nil
}
