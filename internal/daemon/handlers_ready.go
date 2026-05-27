package daemon

import (
	"context"

	"github.com/danielgtaylor/huma/v2"

	"go.kenn.io/kata/internal/api"
)

func registerReadyHandlers(humaAPI huma.API, cfg ServerConfig) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "readyIssues",
		Method:      "GET",
		Path:        "/api/v1/projects/{project_id}/ready",
	}, func(ctx context.Context, in *api.ReadyRequest) (*api.ReadyResponse, error) {
		if _, err := activeProjectByID(ctx, cfg.DB, in.ProjectID); err != nil {
			return nil, err
		}
		issues, err := cfg.DB.ReadyIssues(ctx, in.ProjectID, in.Limit)
		if err != nil {
			return nil, api.NewError(500, "internal", err.Error(), "", nil)
		}
		out := &api.ReadyResponse{}
		out.Body.Issues = issues
		return out, nil
	})

	huma.Register(humaAPI, huma.Operation{
		OperationID: "readyIssuesGlobal",
		Method:      "GET",
		Path:        "/api/v1/ready",
	}, func(ctx context.Context, in *api.ReadyGlobalRequest) (*api.ReadyGlobalResponse, error) {
		issues, err := cfg.DB.ReadyIssuesGlobal(ctx, in.Limit)
		if err != nil {
			return nil, api.NewError(500, "internal", err.Error(), "", nil)
		}
		out := &api.ReadyGlobalResponse{}
		out.Body.Issues = issues
		return out, nil
	})
}
