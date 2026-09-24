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

var explorerViewTypes = []string{"workspaces", "tf_versions", "providers", "modules"}

func QueryExplorer(logger *log.Logger) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool("query_explorer",
			mcp.WithDescription("Query organization-wide Terraform Cloud or Enterprise Explorer data."),
			mcp.WithTitleAnnotation("Query Terraform Explorer"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithString("terraform_org_name", mcp.Required(), mcp.Description("Terraform Cloud or Enterprise organization name")),
			mcp.WithString("view_type", mcp.Required(), mcp.Description("Explorer view to query"), mcp.WithStringEnumItems(explorerViewTypes)),
			mcp.WithString("sort", mcp.Description("Optional snake_case field to sort by; prefix with '-' for descending order")),
			mcp.WithString("fields", mcp.Description("Optional comma-separated list of fields to return")),
			mcp.WithArray("filters", mcp.Description("Optional filters in field:operator:value format"), mcp.WithStringItems()),
			mcp.WithNumber("page_number", mcp.Description("Optional page number"), mcp.Min(1)),
			mcp.WithNumber("page_size", mcp.Description("Optional page size"), mcp.Min(1)),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return queryExplorerHandler(ctx, request, logger)
		},
	}
}

func queryExplorerHandler(ctx context.Context, request mcp.CallToolRequest, logger *log.Logger) (*mcp.CallToolResult, error) {
	organization, err := request.RequireString("terraform_org_name")
	if err != nil {
		return ToolError(logger, "missing required input: terraform_org_name", err)
	}
	viewType, err := request.RequireString("view_type")
	if err != nil {
		return ToolError(logger, "missing required input: view_type", err)
	}

	filters, err := explorerFiltersFromRequest(request)
	if err != nil {
		return ToolError(logger, "invalid filters", err)
	}
	pageNumber := request.GetInt("page_number", 0)
	pageSize := request.GetInt("page_size", 0)
	params := buildExplorerQueryParams(viewType, request.GetString("sort", ""), request.GetString("fields", ""), filters, pageNumber, pageSize)

	explorerClient, err := client.GetExplorerClientFromContext(ctx, logger)
	if err != nil {
		return ToolError(logger, "failed to get Explorer client", err)
	}
	response, err := explorerClient.MakeExplorerCall(ctx, "GET", strings.TrimSpace(organization), "", params, logger)
	if err != nil {
		return ToolError(logger, "failed to query Explorer", err)
	}

	var result client.ExplorerQueryResponse
	if err := json.Unmarshal(response, &result); err != nil {
		return ToolError(logger, "failed to parse Explorer response", err)
	}
	formatted, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return ToolError(logger, "failed to format Explorer response", err)
	}
	return mcp.NewToolResultText(string(formatted)), nil
}

func explorerFiltersFromRequest(request mcp.CallToolRequest) ([]client.ExplorerFilter, error) {
	value, ok := request.GetArguments()["filters"]
	if !ok || value == nil {
		return nil, nil
	}

	var rawFilters []string
	switch filters := value.(type) {
	case []string:
		rawFilters = filters
	case []any:
		for _, filter := range filters {
			filterString, ok := filter.(string)
			if !ok {
				return nil, fmt.Errorf("filter must be a string")
			}
			rawFilters = append(rawFilters, filterString)
		}
	default:
		return nil, fmt.Errorf("filters must be an array of strings")
	}
	return parseExplorerFilters(rawFilters)
}

func parseExplorerFilters(rawFilters []string) ([]client.ExplorerFilter, error) {
	filters := make([]client.ExplorerFilter, 0, len(rawFilters))
	for _, rawFilter := range rawFilters {
		parts := strings.SplitN(rawFilter, ":", 3)
		if len(parts) != 3 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
			return nil, fmt.Errorf("filter %q must use field:operator:value format", rawFilter)
		}
		filters = append(filters, client.ExplorerFilter{
			Field:    strings.TrimSpace(parts[0]),
			Operator: strings.TrimSpace(parts[1]),
			Value:    parts[2],
		})
	}
	return filters, nil
}

func buildExplorerQueryParams(viewType, sort, fields string, filters []client.ExplorerFilter, pageNumber, pageSize int) url.Values {
	params := url.Values{"type": {viewType}}
	if sort != "" {
		params.Set("sort", sort)
	}
	if fields != "" {
		params.Set("fields", fields)
	}
	if pageNumber > 0 {
		params.Set("page[number]", strconv.Itoa(pageNumber))
	}
	if pageSize > 0 {
		params.Set("page[size]", strconv.Itoa(pageSize))
	}
	client.BuildExplorerFilterParams(params, filters)
	return params
}
