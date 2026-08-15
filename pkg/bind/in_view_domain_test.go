package bind

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/biptec/opnsense-go/pkg/api"
)

func TestBindInViewDomainContract(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/bind/domain/add_in_view_domain":
			var body map[string]InViewDomain
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode in-view domain: %v", err)
			}
			got := body["domain"]
			if got.View.String() != "public-view" || got.SourceView.String() != "private-view" || got.DomainName != "acme.biptec.net" {
				t.Fatalf("unexpected in-view domain body: %+v", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"result": "saved", "uuid": "zone-id"})
		case "/api/bind/domain/get_domain/zone-id":
			_ = json.NewEncoder(w).Encode(map[string]any{"domain": map[string]any{
				"enabled": "1", "view": bindSelected("public-view"), "domainname": "acme.biptec.net", "inview": bindSelected("private-view"),
			}})
		case "/api/bind/service/reload":
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	ctx := context.Background()
	controller := bindTestController(server)
	id, err := controller.AddInViewDomain(ctx, &InViewDomain{
		View: api.SelectedMap("public-view"), DomainName: "acme.biptec.net", Enabled: "1", SourceView: api.SelectedMap("private-view"),
	})
	if err != nil || id != "zone-id" {
		t.Fatalf("AddInViewDomain() = %q, %v", id, err)
	}
	zone, err := controller.GetInViewDomain(ctx, id)
	if err != nil || zone.SourceView.String() != "private-view" || zone.View.String() != "public-view" {
		t.Fatalf("GetInViewDomain() = %+v, %v", zone, err)
	}
}
