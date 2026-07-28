package interfaces

import (
	"context"
	"errors"
	"fmt"

	"github.com/biptec/opnsense-go/pkg/api"
)

// AddAssignment creates an assignment and returns the stable logical interface
// identifier assigned by OPNsense (for example, opt1).
//
// The add_item endpoint returns a transient in-memory UUID. During
// reconfiguration OPNsense persists a new assignment under the next available
// logical identifier, so the raw add response UUID must not be returned to
// callers as resource identity.
func (c *Controller) AddAssignment(ctx context.Context, resource *Assignment) (string, error) {
	if resource == nil {
		return "", errors.New("assignment must not be nil")
	}

	device := resource.Device.String()
	if device == "" {
		return "", errors.New("assignment device must not be empty")
	}

	if _, err := api.Add(c.Client(), ctx, AssignmentOpts, resource); err != nil {
		return "", err
	}

	search, err := c.AssignmentSearch(ctx, "-1")
	if err != nil {
		return "", fmt.Errorf("search assignments after create: %w", err)
	}

	for _, row := range search.Rows {
		identifier := row.Identifier
		if identifier == "" {
			identifier = row.UUID
		}
		if identifier == "" {
			continue
		}

		assignment, err := c.GetAssignment(ctx, identifier)
		if err != nil {
			return "", fmt.Errorf("get assignment %q after create: %w", identifier, err)
		}
		if assignment.Device.String() == device {
			return identifier, nil
		}
	}

	return "", fmt.Errorf("created assignment for device %q but could not resolve its logical identifier", device)
}
