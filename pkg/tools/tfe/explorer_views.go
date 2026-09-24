// Copyright IBM Corp. 2025
// SPDX-License-Identifier: MPL-2.0

package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-mcp-server/pkg/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	log "github.com/sirupsen/logrus"
)

func ListExplorerViews(logger *log.Logger) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool("list_explorer_views",
			mcp.WithDescription("List saved Explorer views for a Terraform Cloud or Enterprise organization."),
			mcp.WithTitleAnnotation("List Terraform Explorer views"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithString("terraform_org_name", mcp.Required(), mcp.Description("Terraform Cloud or Enterprise organization name")),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return listExplorerViewsHandler(ctx, request, logger)
		},
	}
}

func GetExplorerView(logger *log.Logger) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool("get_explorer_view",
			mcp.WithDescription("Get a saved Explorer view from a Terraform Cloud or Enterprise organization."),
			mcp.WithTitleAnnotation("Get a Terraform Explorer view"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithString("terraform_org_name", mcp.Required(), mcp.Description("Terraform Cloud or Enterprise organization name")),
			mcp.WithString("view_id", mcp.Required(), mcp.Description("Saved Explorer view ID")),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return getExplorerViewHandler(ctx, request, logger)
		},
	}
}

func QueryExplorerView(logger *log.Logger) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool("query_explorer_view",
			mcp.WithDescription("Re-run a saved Explorer view and return its current results."),
			mcp.WithTitleAnnotation("Query a Terraform Explorer view"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithString("terraform_org_name", mcp.Required(), mcp.Description("Terraform Cloud or Enterprise organization name")),
			mcp.WithString("view_id", mcp.Required(), mcp.Description("Saved Explorer view ID")),
			mcp.WithNumber("page_number", mcp.Description("Optional page number"), mcp.Min(1)),
			mcp.WithNumber("page_size", mcp.Description("Optional page size"), mcp.Min(1)),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return queryExplorerViewHandler(ctx, request, logger)
		},
	}
}

func listExplorerViewsHandler(ctx context.Context, request mcp.CallToolRequest, logger *log.Logger) (*mcp.CallToolResult, error) {
	organization, err := request.RequireString("terraform_org_name")
	if err != nil {
		return ToolError(logger, "missing required input: terraform_org_name", err)
	}
	response, err := makeExplorerViewCall(ctx, logger, organization, "views", nil)
	if err != nil {
		return ToolError(logger, "failed to list Explorer views", err)
	}
	var result client.ExplorerSavedViewsResponse
	if err := json.Unmarshal(response, &result); err != nil {
		return ToolError(logger, "failed to parse Explorer views response", err)
	}
	return formatExplorerJSON(logger, result)
}

func getExplorerViewHandler(ctx context.Context, request mcp.CallToolRequest, logger *log.Logger) (*mcp.CallToolResult, error) {
	organization, viewID, err := explorerViewInputs(request)
	if err != nil {
		return ToolError(logger, "invalid Explorer view input", err)
	}
	response, err := makeExplorerViewCall(ctx, logger, organization, "views/"+url.PathEscape(viewID), nil)
	if err != nil {
		return ToolError(logger, "failed to get Explorer view", err)
	}
	var result client.ExplorerSavedViewResponse
	if err := json.Unmarshal(response, &result); err != nil {
		return ToolError(logger, "failed to parse Explorer view response", err)
	}
	return formatExplorerJSON(logger, result)
}

func queryExplorerViewHandler(ctx context.Context, request mcp.CallToolRequest, logger *log.Logger) (*mcp.CallToolResult, error) {
	organization, viewID, err := explorerViewInputs(request)
	if err != nil {
		return ToolError(logger, "invalid Explorer view input", err)
	}
	params := url.Values{}
	pageNumber := request.GetInt("page_number", 0)
	pageSize := request.GetInt("page_size", 0)
	if pageNumber > 0 {
		params.Set("page[number]", strconv.Itoa(pageNumber))
	}
	if pageSize > 0 {
		params.Set("page[size]", strconv.Itoa(pageSize))
	}
	response, err := makeExplorerViewCall(ctx, logger, organization, "views/"+url.PathEscape(viewID)+"/results", params)
	if err != nil {
		return ToolError(logger, "failed to query Explorer view", err)
	}
	var result client.ExplorerQueryResponse
	if err := json.Unmarshal(response, &result); err != nil {
		return ToolError(logger, "failed to parse Explorer view results", err)
	}
	return formatExplorerJSON(logger, result)
}

func explorerViewInputs(request mcp.CallToolRequest) (string, string, error) {
	organization, err := request.RequireString("terraform_org_name")
	if err != nil {
		return "", "", err
	}
	viewID, err := request.RequireString("view_id")
	if err != nil {
		return "", "", err
	}
	organization = strings.TrimSpace(organization)
	viewID = strings.TrimSpace(viewID)
	if organization == "" || viewID == "" {
		return "", "", fmt.Errorf("organization and view_id cannot be empty")
	}
	return organization, viewID, nil
}

func makeExplorerViewCall(ctx context.Context, logger *log.Logger, organization, subpath string, params url.Values) ([]byte, error) {
	explorerClient, err := client.GetExplorerClientFromContext(ctx, logger)
	if err != nil {
		return nil, err
	}
	return explorerClient.MakeExplorerCall(ctx, "GET", strings.TrimSpace(organization), subpath, params, logger)
}

func formatExplorerJSON(logger *log.Logger, value any) (*mcp.CallToolResult, error) {
	formatted, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return ToolError(logger, "failed to format Explorer response", err)
	}
	return mcp.NewToolResultText(string(formatted)), nil
}
