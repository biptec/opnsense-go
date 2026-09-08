package ndpproxy

import (
	"context"
	"fmt"

	"github.com/biptec/opnsense-go/pkg/api"
)

func (c *Controller) ServiceReconfigure(ctx context.Context) (*api.ReconfigureStatusResult, error) {
	callOpts := api.RPCOpts{
		Endpoint:        api.Endpoint{Path: "/ndpproxy/service/reconfigure", Method: "POST"},
		PathParameters:  []string{},
		QueryParameters: map[string]string{},
		BodyParameters:  map[string]interface{}{},
	}
	resultData := &api.ReconfigureStatusResult{}
	result, err := api.Call(c.Client(), ctx, callOpts, resultData)
	if err != nil {
		return nil, fmt.Errorf("reconfigure call failed: %w", err)
	}
	return result, nil
}
