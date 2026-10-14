// Copyright IBM Corp. 2025
// SPDX-License-Identifier: MPL-2.0

package tools

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-mcp-server/pkg/client"
	"github.com/mark3labs/mcp-go/mcp"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestQueryExplorerGraphTool(t *testing.T) {
	tool := QueryExplorerGraph(log.New())
	require.Equal(t, "query_explorer_graph", tool.Tool.Name)
	require.True(t, *tool.Tool.Annotations.ReadOnlyHint)
	require.False(t, *tool.Tool.Annotations.DestructiveHint)
	require.ElementsMatch(t, []string{"terraform_org_name", "graph_type"}, tool.Tool.InputSchema.Required)

	graphType, ok := tool.Tool.InputSchema.Properties["graph_type"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, []string{"workspace-workspace", "module-workspace", "provider-workspace"}, graphType["enum"])
}

func TestQueryExplorerGraphRejectsUnknownGraphType(t *testing.T) {
	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]any{
		"terraform_org_name": "example-org",
		"graph_type":         "../views",
	}

	result, err := queryExplorerGraphHandler(context.Background(), request, log.New())
	require.NoError(t, err)
	require.True(t, result.IsError)
}

func TestBuildExplorerGraphQueryParams(t *testing.T) {
	params := buildExplorerGraphQueryParams(
		[]client.ExplorerFilter{{Field: "workspace_name", Operator: "contains", Value: "service"}},
		"cursor-123",
	)
	require.Equal(t, "cursor-123", params.Get("page[after]"))
	require.Equal(t, "service", params.Get("filter[0][workspace_name][contains][0]"))

	require.Empty(t, buildExplorerGraphQueryParams(nil, ""))
}

func TestExplorerGraphNextCursor(t *testing.T) {
	cursor, err := explorerGraphNextCursor(nil)
	require.NoError(t, err)
	require.Nil(t, cursor)

	next := "/api/v2/organizations/example-org/explorer/graph/workspace-workspace?filter%5B0%5D%5Bworkspace_name%5D%5Bcontains%5D%5B0%5D=service&page%5Bafter%5D=abc%2B%2F%3D--def"
	cursor, err = explorerGraphNextCursor(&next)
	require.NoError(t, err)
	require.NotNil(t, cursor)
	require.Equal(t, "abc+/=--def", *cursor)

	missing := "/api/v2/organizations/example-org/explorer/graph/workspace-workspace"
	_, err = explorerGraphNextCursor(&missing)
	require.Error(t, err)
}
