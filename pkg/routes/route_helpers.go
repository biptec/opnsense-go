package routes

import (
	"context"
	"strings"
	"time"

	"github.com/biptec/opnsense-go/pkg/api"
	"github.com/biptec/opnsense-go/pkg/routing"
)

type routeGatewayRetrySettings struct {
	routeInterval  time.Duration
	routeTimeout   time.Duration
	verifyInterval time.Duration
	verifyTimeout  time.Duration
}

var defaultRouteGatewayRetrySettings = routeGatewayRetrySettings{
	routeInterval:  time.Second,
	routeTimeout:   25 * time.Second,
	verifyInterval: 250 * time.Millisecond,
	verifyTimeout:  5 * time.Second,
}

// AddRouteResolved tolerates the short configd cache windows after a routing
// gateway is created. It first waits for the gateway to appear in routing
// search, then retries the route option-list validation until it catches up.
func (c *Controller) AddRouteResolved(ctx context.Context, resource *Route) (string, error) {
	return c.addRouteResolvedWithSettings(ctx, resource, defaultRouteGatewayRetrySettings)
}

func (c *Controller) addRouteResolved(
	ctx context.Context,
	resource *Route,
	retryInterval time.Duration,
	retryTimeout time.Duration,
) (string, error) {
	return c.addRouteResolvedWithSettings(ctx, resource, routeGatewayRetrySettings{
		routeInterval: retryInterval,
		routeTimeout:  retryTimeout,
	})
}

func (c *Controller) addRouteResolvedWithSettings(
	ctx context.Context,
	resource *Route,
	settings routeGatewayRetrySettings,
) (string, error) {
	var id string
	err := c.withResolvedGateway(ctx, resource, settings, func() error {
		var err error
		id, err = c.AddRoute(ctx, resource)
		return err
	})
	return id, err
}

// UpdateRouteResolved applies the same stale-gateway protection to updates
// that switch a route to a newly-created gateway.
func (c *Controller) UpdateRouteResolved(ctx context.Context, id string, resource *Route) error {
	return c.updateRouteResolvedWithSettings(ctx, id, resource, defaultRouteGatewayRetrySettings)
}

func (c *Controller) updateRouteResolved(
	ctx context.Context,
	id string,
	resource *Route,
	retryInterval time.Duration,
	retryTimeout time.Duration,
) error {
	return c.updateRouteResolvedWithSettings(ctx, id, resource, routeGatewayRetrySettings{
		routeInterval: retryInterval,
		routeTimeout:  retryTimeout,
	})
}

func (c *Controller) updateRouteResolvedWithSettings(
	ctx context.Context,
	id string,
	resource *Route,
	settings routeGatewayRetrySettings,
) error {
	return c.withResolvedGateway(ctx, resource, settings, func() error {
		return c.UpdateRoute(ctx, id, resource)
	})
}

func (c *Controller) withResolvedGateway(
	ctx context.Context,
	resource *Route,
	settings routeGatewayRetrySettings,
	action func() error,
) error {
	deadline := time.Now().Add(settings.routeTimeout)
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
			valid, verifyErr := c.waitForRouteGateway(ctx, resource, settings.verifyInterval, settings.verifyTimeout)
			if verifyErr != nil {
				return verifyErr
			}
			if !valid {
				return err
			}
			gatewayVerified = true
		}
		if settings.routeTimeout <= 0 || time.Now().Add(settings.routeInterval).After(deadline) {
			return err
		}

		timer := time.NewTimer(settings.routeInterval)
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

func (c *Controller) waitForRouteGateway(
	ctx context.Context,
	resource *Route,
	retryInterval time.Duration,
	retryTimeout time.Duration,
) (bool, error) {
	deadline := time.Now().Add(retryTimeout)
	for {
		exists, err := c.routeGatewayExists(ctx, resource)
		if err != nil || exists {
			return exists, err
		}
		if retryTimeout <= 0 || time.Now().Add(retryInterval).After(deadline) {
			return false, nil
		}

		timer := time.NewTimer(retryInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return false, ctx.Err()
		case <-timer.C:
		}
	}
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

	type gatewayRouteRow struct {
		Name       string `json:"name"`
		IPProtocol string `json:"ipprotocol"`
		Gateway    string `json:"gateway"`
	}
	result, err := api.Search[gatewayRouteRow](c.Client(), ctx, routing.GatewayOpts.Search)
	if err != nil {
		return false, err
	}
	for _, gateway := range result.Rows {
		if gateway.Name != name {
			continue
		}
		protocol := gateway.IPProtocol
		if protocol == "" && gateway.Gateway != "" {
			protocol = "inet"
			if strings.Contains(gateway.Gateway, ":") {
				protocol = "inet6"
			}
		}
		if protocol == expectedProtocol || (expectedProtocol == "inet" && protocol == "inet6") {
			return true, nil
		}
		return false, nil
	}
	return false, nil
}
