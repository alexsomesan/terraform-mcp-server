// Copyright IBM Corp. 2025
// SPDX-License-Identifier: MPL-2.0

package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-mcp-server/pkg/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	log "github.com/sirupsen/logrus"
)

var explorerGraphTypes = []string{"workspace-workspace", "module-workspace", "provider-workspace"}

type explorerGraphResult struct {
	Data          client.ExplorerGraph `json:"data"`
	NextPageAfter *string              `json:"next_page_after"`
}

func QueryExplorerGraph(logger *log.Logger) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool("query_explorer_graph",
			mcp.WithDescription(`Get an organization-wide relationship graph (nodes and edges) from Terraform Cloud or Enterprise Explorer.
Edges point from a dependency to the workspace that depends on it:
- workspace-workspace: producer workspace -> workspace that consumes its state outputs
- module-workspace: module version -> workspace that uses it
- provider-workspace: provider version -> workspace that uses it
Results are paginated. While next_page_after is not null, call again with it as page_after and the same filters; pages may be empty. Merge nodes and edges across pages by id.`),
			mcp.WithTitleAnnotation("Query a Terraform Explorer relationship graph"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithString("terraform_org_name", mcp.Required(), mcp.Description("Terraform Cloud or Enterprise organization name")),
			mcp.WithString("graph_type", mcp.Required(), mcp.Description("Relationship graph to query"), mcp.Enum(explorerGraphTypes...)),
			mcp.WithArray("filters", mcp.Description("Optional filters in field:operator:value format, applied to the graph's root entity (workspaces, modules or providers) with the same fields as query_explorer"), mcp.WithStringItems()),
			mcp.WithString("page_after", mcp.Description("Optional cursor from a previous response's next_page_after")),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return queryExplorerGraphHandler(ctx, request, logger)
		},
	}
}

func queryExplorerGraphHandler(ctx context.Context, request mcp.CallToolRequest, logger *log.Logger) (*mcp.CallToolResult, error) {
	organization, err := request.RequireString("terraform_org_name")
	if err != nil {
		return ToolError(logger, "missing required input: terraform_org_name", err)
	}
	graphType, err := request.RequireString("graph_type")
	if err != nil {
		return ToolError(logger, "missing required input: graph_type", err)
	}
	graphType = strings.TrimSpace(graphType)
	if !slices.Contains(explorerGraphTypes, graphType) {
		return ToolErrorf(logger, "invalid graph_type %q: must be one of %s", graphType, strings.Join(explorerGraphTypes, ", "))
	}

	filters, err := explorerFiltersFromRequest(request)
	if err != nil {
		return ToolError(logger, "invalid filters", err)
	}
	params := buildExplorerGraphQueryParams(filters, strings.TrimSpace(request.GetString("page_after", "")))

	response, err := makeExplorerViewCall(ctx, logger, organization, "graph/"+graphType, params)
	if err != nil {
		return ToolError(logger, "failed to query Explorer graph", err)
	}
	var graph client.ExplorerGraphResponse
	if err := json.Unmarshal(response, &graph); err != nil {
		return ToolError(logger, "failed to parse Explorer graph response", err)
	}
	nextPageAfter, err := explorerGraphNextCursor(graph.Links.Next)
	if err != nil {
		return ToolError(logger, "failed to parse Explorer graph next link", err)
	}
	return formatExplorerJSON(logger, explorerGraphResult{Data: graph.Data, NextPageAfter: nextPageAfter})
}

func buildExplorerGraphQueryParams(filters []client.ExplorerFilter, pageAfter string) url.Values {
	params := url.Values{}
	if pageAfter != "" {
		params.Set("page[after]", pageAfter)
	}
	client.BuildExplorerFilterParams(params, filters)
	return params
}

func explorerGraphNextCursor(next *string) (*string, error) {
	if next == nil || *next == "" {
		return nil, nil
	}
	nextURL, err := url.Parse(*next)
	if err != nil {
		return nil, err
	}
	cursor := nextURL.Query().Get("page[after]")
	if cursor == "" {
		return nil, fmt.Errorf("next link has no page[after] cursor")
	}
	return &cursor, nil
}
