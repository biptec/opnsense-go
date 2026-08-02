package caddy

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/biptec/opnsense-go/pkg/api"
)

func TestCaddyAcceptance(t *testing.T) {
	uri, key, secret := os.Getenv("OPNSENSE_URI"), os.Getenv("OPNSENSE_API_KEY"), os.Getenv("OPNSENSE_API_SECRET")
	if uri == "" || key == "" || secret == "" {
		t.Skip("live OPNsense credentials are not configured")
	}

	controller := &Controller{Api: api.NewClient(api.Options{
		Uri: uri, APIKey: key, APISecret: secret, AllowInsecure: true,
		MaxBackoff: 10, MinBackoff: 1, MaxRetries: 2,
	})}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	settings, err := controller.SettingsGet(ctx)
	if err != nil {
		if strings.Contains(err.Error(), "status code non-200") {
			t.Skipf("os-caddy is not installed on this test image: %v", err)
		}
		t.Fatalf("SettingsGet(): %v", err)
	}
	if _, err := controller.SettingsSet(ctx, &settings.Caddy); err != nil {
		t.Fatalf("SettingsSet(): %v", err)
	}

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	var accessID, domainID, handlerID string
	t.Cleanup(func() {
		cleanup, done := context.WithTimeout(context.Background(), time.Minute)
		defer done()
		if handlerID != "" {
			_ = controller.DeleteHandler(cleanup, handlerID)
		}
		if domainID != "" {
			_ = controller.DeleteDomain(cleanup, domainID)
		}
		if accessID != "" {
			_ = controller.DeleteAccessList(cleanup, accessID)
		}
	})

	accessID, err = controller.AddAccessList(ctx, &AccessList{
		Name:           "go-acceptance-" + suffix,
		ClientIPs:      api.SelectedMapList{"192.0.2.0/24"},
		RequestMatcher: api.SelectedMap("client_ip"),
		Description:    "opnsense-go acceptance test",
	})
	if err != nil {
		t.Fatalf("AddAccessList(): %v", err)
	}
	access, err := controller.GetAccessList(ctx, accessID)
	if err != nil || access.Name == "" {
		t.Fatalf("GetAccessList() = %+v, %v", access, err)
	}
	access.ClientIPs = api.SelectedMapList{"192.0.2.0/24", "198.51.100.10"}
	if err := controller.UpdateAccessList(ctx, accessID, access); err != nil {
		t.Fatalf("UpdateAccessList(): %v", err)
	}

	domainID, err = controller.AddDomain(ctx, &Domain{
		Enabled: "1", Domain: "caddy-" + suffix + ".invalid", DisableTLS: api.SelectedMap("1"),
		DNSChallenge: "0", DynamicDNS: "0", AccessLog: "0",
		Description: "opnsense-go acceptance test",
	})
	if err != nil {
		t.Fatalf("AddDomain(): %v", err)
	}
	domain, err := controller.GetDomain(ctx, domainID)
	if err != nil || domain.DisableTLS.String() != "1" {
		t.Fatalf("GetDomain() = %+v, %v", domain, err)
	}

	handlerID, err = controller.AddHandler(ctx, &Handler{
		Enabled: "1", Domain: api.SelectedMap(domainID), Type: api.SelectedMap("handle"),
		Directive: api.SelectedMap("reverse_proxy"), UpstreamDomains: api.SelectedMapList{"127.0.0.1"},
		UpstreamPort: "8080", UpstreamProtocol: api.SelectedMap("0"),
		Description: "opnsense-go acceptance test",
	})
	if err != nil {
		t.Fatalf("AddHandler(): %v", err)
	}
	handler, err := controller.GetHandler(ctx, handlerID)
	if err != nil || handler.Domain.String() != domainID {
		t.Fatalf("GetHandler() = %+v, %v", handler, err)
	}

	domains, err := controller.SearchDomain(ctx)
	if err != nil || domains.Total < 1 {
		t.Fatalf("SearchDomain() = %+v, %v", domains, err)
	}
	handlers, err := controller.SearchHandler(ctx)
	if err != nil || handlers.Total < 1 {
		t.Fatalf("SearchHandler() = %+v, %v", handlers, err)
	}
	accessLists, err := controller.SearchAccessList(ctx)
	if err != nil || accessLists.Total < 1 {
		t.Fatalf("SearchAccessList() = %+v, %v", accessLists, err)
	}
}
