package core

import (
	"context"
	"fmt"
	"strings"

	"github.com/biptec/opnsense-go/pkg/api"
)

// HasyncSyncResult is returned by the native OPNsense selective HA synchronization endpoint.
type HasyncSyncResult struct {
	Status  string   `json:"status"`
	Message string   `json:"message,omitempty"`
	Items   []string `json:"items,omitempty"`
}

// HasyncSync synchronizes only the requested HA configuration items.
// OPNsense validates that every requested item is enabled in High Availability settings.
func (c *Controller) HasyncSync(ctx context.Context, items []string) (*HasyncSyncResult, error) {
	normalized := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			return nil, fmt.Errorf("HA synchronization items must not contain empty values")
		}
		if _, exists := seen[item]; exists {
			continue
		}
		seen[item] = struct{}{}
		normalized = append(normalized, item)
	}
	if len(normalized) == 0 {
		return nil, fmt.Errorf("at least one HA synchronization item is required")
	}

	callOpts := api.RPCOpts{
		Endpoint: api.Endpoint{Path: "/core/hasync_status/sync", Method: "POST"},
		BodyParameters: map[string]interface{}{
			"items": normalized,
		},
	}
	result, err := api.Call(c.Client(), ctx, callOpts, &HasyncSyncResult{})
	if err != nil {
		return nil, fmt.Errorf("HA synchronization call failed: %w", err)
	}
	if result.Status != "ok" {
		if result.Message != "" {
			return nil, fmt.Errorf("HA synchronization failed: %s", result.Message)
		}
		return nil, fmt.Errorf("HA synchronization failed with status %q", result.Status)
	}
	return result, nil
}
