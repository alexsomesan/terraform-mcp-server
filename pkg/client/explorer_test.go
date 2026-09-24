// Copyright IBM Corp. 2025
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestMakeExplorerCall(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/v2/organizations/example-org/explorer", r.URL.Path)
		require.Equal(t, "Bearer secret", r.Header.Get("Authorization"))
		require.Equal(t, "test-client", r.Header.Get("X-Forwarded-For"))
		require.Equal(t, "workspaces", r.URL.Query().Get("type"))
		require.Equal(t, "test", r.URL.Query().Get("filter[0][workspace_name][contains][0]"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer server.Close()

	params := url.Values{"type": {"workspaces"}}
	BuildExplorerFilterParams(params, []ExplorerFilter{{Field: "workspace_name", Operator: "contains", Value: "test"}})
	client := &ExplorerClient{HTTPClient: server.Client(), Address: server.URL, Token: "secret", ClientIP: "test-client"}

	body, err := client.MakeExplorerCall(context.Background(), http.MethodGet, "example-org", "", params, log.New())
	require.NoError(t, err)
	require.JSONEq(t, `{"data":[]}`, string(body))
}

func TestMakeExplorerCallErrors(t *testing.T) {
	for _, status := range []int{http.StatusNotFound, http.StatusTooManyRequests, http.StatusInternalServerError} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
				_, _ = w.Write([]byte(`{"errors":[{"detail":"denied"}]}`))
			}))
			defer server.Close()

			client := &ExplorerClient{HTTPClient: server.Client(), Address: server.URL, Token: "secret"}
			_, err := client.MakeExplorerCall(context.Background(), http.MethodGet, "example-org", "", nil, log.New())
			require.Error(t, err)
			require.True(t, strings.Contains(err.Error(), "denied"))
		})
	}
}
