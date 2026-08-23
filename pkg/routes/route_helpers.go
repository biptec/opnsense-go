package routes

import (
	"context"
	"strings"
	"time"

	"github.com/biptec/opnsense-go/pkg/routing"
)

const (
	routeGatewayRetryInterval = time.Second
	routeGatewayRetryTimeout  = 25 * time.Second
)

// AddRouteResolved tolerates the short configd cache window after a routing
// gateway is created. It retries only when the route gateway validation is
// stale and the referenced gateway already exists with a compatible family.
func (c *Controller) AddRouteResolved(ctx context.Context, resource *Route) (string, error) {
	return c.addRouteResolved(ctx, resource, routeGatewayRetryInterval, routeGatewayRetryTimeout)
}

func (c *Controller) addRouteResolved(
	ctx context.Context,
	resource *Route,
	retryInterval time.Duration,
	retryTimeout time.Duration,
) (string, error) {
	var id string
	err := c.withResolvedGateway(ctx, resource, retryInterval, retryTimeout, func() error {
		var err error
		id, err = c.AddRoute(ctx, resource)
		return err
	})
	return id, err
}

// UpdateRouteResolved applies the same stale-gateway protection to updates
// that switch a route to a newly-created gateway.
func (c *Controller) UpdateRouteResolved(ctx context.Context, id string, resource *Route) error {
	return c.updateRouteResolved(ctx, id, resource, routeGatewayRetryInterval, routeGatewayRetryTimeout)
}

func (c *Controller) updateRouteResolved(
	ctx context.Context,
	id string,
	resource *Route,
	retryInterval time.Duration,
	retryTimeout time.Duration,
) error {
	return c.withResolvedGateway(ctx, resource, retryInterval, retryTimeout, func() error {
		return c.UpdateRoute(ctx, id, resource)
	})
}

func (c *Controller) withResolvedGateway(
	ctx context.Context,
	resource *Route,
	retryInterval time.Duration,
	retryTimeout time.Duration,
	action func() error,
) error {
	deadline := time.Now().Add(retryTimeout)
	gatewayVerified := false
	for {
		err := action()
		if err == nil {
			return nil
		}
		if !isStaleRouteGatewayError(err) {
			return err
		}

		if !gatewayVerified {
			valid, verifyErr := c.routeGatewayExists(ctx, resource)
			if verifyErr != nil {
				return verifyErr
			}
			if !valid {
				return err
			}
			gatewayVerified = true
		}
		if retryTimeout <= 0 || time.Now().Add(retryInterval).After(deadline) {
			return err
		}

		timer := time.NewTimer(retryInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func isStaleRouteGatewayError(err error) bool {
	message := err.Error()
	return strings.Contains(message, "route.gateway:") &&
		strings.Contains(message, "Specify a valid gateway from the list matching the networks ip protocol.")
}

func (c *Controller) routeGatewayExists(ctx context.Context, resource *Route) (bool, error) {
	name := resource.Gateway.String()
	if name == "" {
		return false, nil
	}

	expectedProtocol := "inet"
	if strings.Contains(resource.Network, ":") {
		expectedProtocol = "inet6"
	}

	controller := routing.Controller{Api: c.Client()}
	result, err := controller.SearchGateway(ctx)
	if err != nil {
		return false, err
	}
	for _, gateway := range result.Rows {
		if gateway.Name != name {
			continue
		}
		protocol := gateway.IPProtocol.String()
		if protocol == expectedProtocol || (expectedProtocol == "inet" && protocol == "inet6") {
			return true, nil
		}
		return false, nil
	}
	return false, nil
}
