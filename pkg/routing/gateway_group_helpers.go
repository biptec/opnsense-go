package routing

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const (
	gatewayGroupRetryInterval = time.Second
	gatewayGroupRetryTimeout  = 25 * time.Second
)

// AddGatewayGroupResolved tolerates the short configd cache window used by
// OPNsense's gateway option list. It retries only when the referenced gateways
// already exist and validation failed because that cached list is stale.
func (c *Controller) AddGatewayGroupResolved(ctx context.Context, resource *GatewayGroup) (string, error) {
	return c.addGatewayGroupResolved(ctx, resource, gatewayGroupRetryInterval, gatewayGroupRetryTimeout)
}

func (c *Controller) addGatewayGroupResolved(
	ctx context.Context,
	resource *GatewayGroup,
	retryInterval time.Duration,
	retryTimeout time.Duration,
) (string, error) {
	deadline := time.Now().Add(retryTimeout)
	for {
		id, err := c.AddGatewayGroup(ctx, resource)
		if err == nil {
			return id, nil
		}
		if !isStaleGatewayOptionError(err) {
			return "", err
		}

		exist, verifyErr := c.gatewayGroupMembersExist(ctx, resource)
		if verifyErr != nil {
			return "", fmt.Errorf("verify gateway group members: %w", verifyErr)
		}
		if !exist || retryTimeout <= 0 || time.Now().Add(retryInterval).After(deadline) {
			return "", err
		}

		timer := time.NewTimer(retryInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return "", ctx.Err()
		case <-timer.C:
		}
	}
}

func isStaleGatewayOptionError(err error) bool {
	message := err.Error()
	return strings.Contains(message, "Option [") && strings.Contains(message, "not in list")
}

func (c *Controller) gatewayGroupMembersExist(ctx context.Context, resource *GatewayGroup) (bool, error) {
	wanted := gatewayGroupMemberNames(resource)
	if len(wanted) == 0 {
		return false, nil
	}

	result, err := c.SearchGateway(ctx)
	if err != nil {
		return false, err
	}
	for _, gateway := range result.Rows {
		delete(wanted, gateway.Name)
	}
	return len(wanted) == 0, nil
}

func gatewayGroupMemberNames(resource *GatewayGroup) map[string]struct{} {
	result := make(map[string]struct{})
	for _, tier := range [][]string{resource.Tier1, resource.Tier2, resource.Tier3, resource.Tier4, resource.Tier5} {
		for _, name := range tier {
			if name != "" {
				result[name] = struct{}{}
			}
		}
	}
	return result
}
