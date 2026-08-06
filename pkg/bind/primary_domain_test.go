package bind

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/biptec/opnsense-go/pkg/api"
)

func TestPrimaryDomainTransferContract(t *testing.T) {
	t.Parallel()

	reconfigureCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/bind/domain/add_primary_domain":
			if r.Method != http.MethodPost {
				t.Fatalf("add primary domain method = %s, want POST", r.Method)
			}
			var body map[string]PrimaryDomain
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode primary domain: %v", err)
			}
			domain := body["domain"]
			if domain.TransferKey.String() != "transfer-key-id" || domain.AlsoNotify.String() != "192.0.2.54,192.0.2.55" {
				t.Fatalf("unexpected primary transfer fields: %+v", domain)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"result": "saved", "uuid": "zone-id"})
		case "/api/bind/domain/get_domain/zone-id":
			if r.Method != http.MethodGet {
				t.Fatalf("get primary domain method = %s, want GET", r.Method)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"domain": map[string]any{
				"view":               bindSelected("view-id"),
				"domainname":         "example.net",
				"enabled":            "1",
				"allowtransfer":      map[string]any{},
				"allowrndctransfer":  "0",
				"primarytransferkey": bindSelected("transfer-key-id"),
				"alsonotify":         bindSelectedList("192.0.2.54", "192.0.2.55"),
				"allowquery":         map[string]any{},
				"allowrndcupdate":    "0",
				"updatekeys":         map[string]any{},
				"updatepolicy":       bindSelected("self_txt"),
				"dnssec":             "0",
				"serial":             "2026080601",
				"ttl":                "300",
				"refresh":            "3600",
				"retry":              "600",
				"expire":             "1209600",
				"negative":           "300",
				"mailadmin":          "hostmaster@example.net",
				"dnsserver":          "ns1.example.net",
			}})
		case "/api/bind/service/reconfigure":
			reconfigureCalls++
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	controller := bindTestController(server)
	ctx := context.Background()
	id, err := controller.AddPrimaryDomain(ctx, &PrimaryDomain{
		View:              api.SelectedMap("view-id"),
		DomainName:        "example.net",
		Enabled:           "1",
		AllowRndcTransfer: "0",
		TransferKey:       api.SelectedMap("transfer-key-id"),
		AlsoNotify:        api.SelectedMapList{"192.0.2.54", "192.0.2.55"},
		AllowRndcUpdate:   "0",
		UpdatePolicy:      api.SelectedMap("self_txt"),
		DNSSEC:            "0",
		TimeToLive:        "300",
		Refresh:           "3600",
		Retry:             "600",
		Expire:            "1209600",
		Negative:          "300",
		MailAdmin:         "hostmaster@example.net",
		DnsServer:         "ns1.example.net",
	})
	if err != nil || id != "zone-id" {
		t.Fatalf("AddPrimaryDomain() = %q, %v", id, err)
	}

	domain, err := controller.GetPrimaryDomain(ctx, id)
	if err != nil {
		t.Fatalf("GetPrimaryDomain(): %v", err)
	}
	if domain.TransferKey.String() != "transfer-key-id" || domain.AlsoNotify.String() != "192.0.2.54,192.0.2.55" {
		t.Fatalf("unexpected primary transfer response: %+v", domain)
	}
	if reconfigureCalls != 1 {
		t.Fatalf("reconfigure calls = %d, want 1", reconfigureCalls)
	}
}

func bindSelectedList(values ...string) map[string]any {
	result := make(map[string]any, len(values))
	for _, value := range values {
		result[value] = map[string]any{"value": value, "selected": true}
	}
	return result
}
