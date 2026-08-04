package api

import "context"

// SearchResponse is the standard response returned by OPNsense searchBase
// controller actions.
type SearchResponse[T any] struct {
	Total    int `json:"total"`
	RowCount int `json:"rowCount"`
	Current  int `json:"current"`
	Rows     []T `json:"rows"`
}

// Search calls an OPNsense searchBase endpoint and decodes its paginated rows.
func Search[T any](c *Client, ctx context.Context, endpoint Endpoint) (*SearchResponse[T], error) {
	result := &SearchResponse[T]{}
	if err := c.doEndpointRequest(ctx, endpoint, nil, result); err != nil {
		return nil, err
	}
	return result, nil
}
