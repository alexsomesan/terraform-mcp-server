// Copyright IBM Corp. 2025
// SPDX-License-Identifier: MPL-2.0

package tools

import (
	"testing"

	"github.com/hashicorp/terraform-mcp-server/pkg/client"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestQueryExplorerTool(t *testing.T) {
	tool := QueryExplorer(log.New())
	require.Equal(t, "query_explorer", tool.Tool.Name)
	require.True(t, *tool.Tool.Annotations.ReadOnlyHint)
	require.False(t, *tool.Tool.Annotations.DestructiveHint)
	require.Contains(t, tool.Tool.InputSchema.Required, "terraform_org_name")
	require.Contains(t, tool.Tool.InputSchema.Required, "view_type")
}

func TestParseExplorerFilters(t *testing.T) {
	filters, err := parseExplorerFilters([]string{"workspace_name:contains:payments", "workspace_count:gt:10"})
	require.NoError(t, err)
	require.Equal(t, []client.ExplorerFilter{
		{Field: "workspace_name", Operator: "contains", Value: "payments"},
		{Field: "workspace_count", Operator: "gt", Value: "10"},
	}, filters)

	_, err = parseExplorerFilters([]string{"workspace_name:contains"})
	require.Error(t, err)
}

func TestBuildExplorerQueryParams(t *testing.T) {
	params := buildExplorerQueryParams(
		"workspaces",
		"-workspace_name",
		"workspace_name,drifted",
		[]client.ExplorerFilter{{Field: "workspace_name", Operator: "contains", Value: "test"}},
		2,
		25,
	)
	require.Equal(t, "workspaces", params.Get("type"))
	require.Equal(t, "-workspace_name", params.Get("sort"))
	require.Equal(t, "workspace_name,drifted", params.Get("fields"))
	require.Equal(t, "2", params.Get("page[number]"))
	require.Equal(t, "25", params.Get("page[size]"))
	require.Equal(t, "test", params.Get("filter[0][workspace_name][contains][0]"))
}
