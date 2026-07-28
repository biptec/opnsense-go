package interfaces

import (
	"context"
	"fmt"
)

// AddAssignmentResolved creates an interface assignment and returns its stable
// logical identifier (for example opt1). AssignmentController creates the
// persistent identifier during model serialization, after addBase has already
// selected its temporary node UUID, so the UUID in the add response cannot be
// used for a subsequent get. Device assignment is unique in OPNsense, which
// makes a post-create search deterministic.
func (c *Controller) AddAssignmentResolved(ctx context.Context, resource *Assignment) (string, error) {
	if _, err := c.AddAssignment(ctx, resource); err != nil {
		return "", err
	}

	results, err := c.SearchAssignment(ctx)
	if err != nil {
		return "", fmt.Errorf("resolve created assignment: %w", err)
	}

	var identifier string
	for _, candidate := range results.Rows {
		if candidate.Device.String() != resource.Device.String() {
			continue
		}
		current := candidate.Identifier
		if current == "" {
			current = candidate.UUID
		}
		if current == "" {
			continue
		}
		if identifier != "" && identifier != current {
			return "", fmt.Errorf("resolve created assignment: multiple assignments found for device %q", resource.Device.String())
		}
		identifier = current
	}

	if identifier == "" {
		return "", fmt.Errorf("resolve created assignment: no assignment found for device %q", resource.Device.String())
	}
	return identifier, nil
}
