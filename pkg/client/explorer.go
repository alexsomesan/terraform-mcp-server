// Copyright IBM Corp. 2025
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/hashicorp/terraform-mcp-server/pkg/utils"
	"github.com/hashicorp/terraform-mcp-server/version"
	log "github.com/sirupsen/logrus"
)

type ExplorerClient struct {
	HTTPClient *http.Client
	Address    string
	Token      string
	ClientIP   string
}

type ExplorerFilter struct {
	Field    string
	Operator string
	Value    string
}

type ExplorerRow struct {
	ID         string         `json:"id"`
	Type       string         `json:"type"`
	Attributes map[string]any `json:"attributes"`
}

type ExplorerQueryResponse struct {
	Data  []ExplorerRow        `json:"data"`
	Links map[string]any       `json:"links"`
	Meta  ExplorerResponseMeta `json:"meta"`
}

type ExplorerResponseMeta struct {
	Pagination ExplorerPagination `json:"pagination"`
}

type ExplorerPagination struct {
	CurrentPage int  `json:"current-page"`
	PageSize    int  `json:"page-size"`
	NextPage    *int `json:"next-page"`
	PrevPage    *int `json:"prev-page"`
	TotalPages  int  `json:"total-pages"`
	TotalCount  int  `json:"total-count"`
}

type ExplorerSavedView struct {
	ID         string                      `json:"id"`
	Type       string                      `json:"type"`
	Attributes ExplorerSavedViewAttributes `json:"attributes"`
}

type ExplorerSavedViewAttributes struct {
	Name      string          `json:"name"`
	CreatedAt string          `json:"created-at"`
	Query     json.RawMessage `json:"query"`
	QueryType string          `json:"query-type"`
}

type ExplorerSavedViewsResponse struct {
	Data []ExplorerSavedView `json:"data"`
}

type ExplorerSavedViewResponse struct {
	Data ExplorerSavedView `json:"data"`
}

func GetExplorerClientFromContext(ctx context.Context, logger *log.Logger) (*ExplorerClient, error) {
	httpClient, err := GetHttpClientFromContext(ctx, logger)
	if err != nil {
		return nil, err
	}

	address, _ := ctx.Value(contextKey(TerraformAddress)).(string)
	if address == "" {
		address = utils.GetEnv(TerraformAddress, DefaultTerraformAddress)
	}
	token, _ := ctx.Value(contextKey(TerraformToken)).(string)
	if token == "" {
		token = utils.GetEnv(TerraformToken, "")
	}
	if token == "" {
		return nil, fmt.Errorf("no Terraform token provided")
	}

	clientIP, _ := ctx.Value(contextKey(ClientIPKey)).(string)
	return &ExplorerClient{HTTPClient: httpClient, Address: strings.TrimRight(address, "/"), Token: token, ClientIP: clientIP}, nil
}

func (c *ExplorerClient) MakeExplorerCall(ctx context.Context, method, organization, subpath string, query url.Values, logger *log.Logger) ([]byte, error) {
	if c == nil || c.HTTPClient == nil {
		return nil, fmt.Errorf("Explorer HTTP client is not configured")
	}
	if c.Address == "" || c.Token == "" {
		return nil, fmt.Errorf("Explorer client address and token are required")
	}

	endpoint, err := url.Parse(c.Address)
	if err != nil {
		return nil, fmt.Errorf("parsing Terraform address: %w", err)
	}
	endpoint.Path = path.Join(endpoint.Path, "api", "v2", "organizations", organization, "explorer", subpath)
	if query != nil {
		endpoint.RawQuery = query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("User-Agent", fmt.Sprintf("terraform-mcp-server/%s", version.GetHumanVersion()))
	if c.ClientIP != "" {
		req.Header.Set("X-Forwarded-For", c.ClientIP)
	}

	logger.Debugf("Requested Explorer URL: %s", endpoint)
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		message := strings.TrimSpace(string(body))
		if message == "" {
			message = resp.Status
		}
		return nil, fmt.Errorf("Explorer API request failed: %s", message)
	}

	return body, nil
}

func BuildExplorerFilterParams(params url.Values, filters []ExplorerFilter) {
	for index, filter := range filters {
		params.Set(fmt.Sprintf("filter[%d][%s][%s][0]", index, filter.Field, filter.Operator), filter.Value)
	}
}
