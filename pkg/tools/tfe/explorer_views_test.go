// Copyright IBM Corp. 2025
// SPDX-License-Identifier: MPL-2.0

package tools

import (
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestExplorerViewTools(t *testing.T) {
	logger := log.New()
	for _, tool := range []struct {
		name string
		tool string
	}{
		{name: "list", tool: "list_explorer_views"},
		{name: "get", tool: "get_explorer_view"},
		{name: "query", tool: "query_explorer_view"},
	} {
		t.Run(tool.name, func(t *testing.T) {
			var actualName string
			switch tool.tool {
			case "list_explorer_views":
				actualName = ListExplorerViews(logger).Tool.Name
			case "get_explorer_view":
				actualName = GetExplorerView(logger).Tool.Name
			case "query_explorer_view":
				actualName = QueryExplorerView(logger).Tool.Name
			}
			require.Equal(t, tool.tool, actualName)
		})
	}
}
