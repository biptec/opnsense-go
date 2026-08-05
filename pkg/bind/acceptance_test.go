package bind

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/biptec/opnsense-go/pkg/api"
)

func availableViewSequence(views []View) (string, error) {
	used := make(map[int]bool)
	upperBound := 10000
	for _, view := range views {
		sequence, err := strconv.Atoi(view.Sequence)
		if err != nil {
			continue
		}
		used[sequence] = true
		if view.MatchAny == "1" && sequence < upperBound {
			upperBound = sequence
		}
	}
	for sequence := 1; sequence < upperBound; sequence++ {
		if !used[sequence] {
			return strconv.Itoa(sequence), nil
		}
	}
	return "", fmt.Errorf("no sequence is available before the catch-all view")
}

func TestBindAcceptance(t *testing.T) {
	uri, key, secret := os.Getenv("OPNSENSE_URI"), os.Getenv("OPNSENSE_API_KEY"), os.Getenv("OPNSENSE_API_SECRET")
	if uri == "" || key == "" || secret == "" {
		t.Skip("live OPNsense credentials are not configured")
	}

	controller := &Controller{Api: api.NewClient(api.Options{
		Uri: uri, APIKey: key, APISecret: secret, AllowInsecure: true,
		MaxBackoff: 10, MinBackoff: 1, MaxRetries: 2,
	})}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	settings, err := controller.SettingsGet(ctx)
	if err != nil {
		t.Fatalf("SettingsGet(): %v", err)
	}
	if settings.General.Port == "" {
		t.Fatalf("SettingsGet() returned no listen port: %+v", settings.General)
	}
	if _, err := controller.SettingsSet(ctx, &settings.General); err != nil {
		t.Fatalf("SettingsSet(): %v", err)
	}
	if _, err := controller.ServiceStatus(ctx); err != nil {
		t.Fatalf("ServiceStatus(): %v", err)
	}

	views, err := controller.SearchView(ctx)
	if err != nil {
		t.Fatalf("SearchView(): %v", err)
	}
	sequence, err := availableViewSequence(views.Rows)
	if err != nil {
		t.Fatal(err)
	}

	suffix := strconv.FormatInt(time.Now().UnixNano(), 36)
	name := "go-bind-" + suffix
	domainName := name + ".invalid"
	var aclID, viewID, keyID, domainID string
	var recordIDs []string
	t.Cleanup(func() {
		cleanup, done := context.WithTimeout(context.Background(), 2*time.Minute)
		defer done()
		for index := len(recordIDs) - 1; index >= 0; index-- {
			_ = controller.DeleteRecord(cleanup, recordIDs[index])
		}
		if domainID != "" {
			_ = controller.DeletePrimaryDomain(cleanup, domainID)
		}
		if keyID != "" {
			_ = controller.DeleteTsigKey(cleanup, keyID)
		}
		if viewID != "" {
			_ = controller.DeleteView(cleanup, viewID)
		}
		if aclID != "" {
			_ = controller.DeleteAcl(cleanup, aclID)
		}
	})

	aclID, err = controller.AddAcl(ctx, &Acl{
		Enabled: "1", Name: name, Networks: api.SelectedMapList{"198.51.100.0/24"},
	})
	if err != nil {
		t.Fatalf("AddAcl(%q): %v", aclID, err)
	}

	viewID, err = controller.AddView(ctx, &View{
		Enabled: "1", Sequence: sequence, Name: name, MatchAny: "0",
		MatchClients: api.SelectedMapList{aclID}, Recursion: "0", AllowQueryAny: "0",
		AllowQuery: api.SelectedMapList{aclID}, DNSSECValidation: api.SelectedMap("auto"),
	})
	if err != nil {
		t.Fatalf("AddView(%q): %v", viewID, err)
	}

	generated, err := controller.TSIGGenerateSecret(ctx)
	if err != nil {
		t.Fatalf("TSIGGenerateSecret(): %v", err)
	}
	if generated.Secret == "" {
		t.Fatal("TSIGGenerateSecret() returned an empty secret")
	}
	keyID, err = controller.AddTsigKey(ctx, &TsigKey{
		Enabled: "1", Name: name, Algorithm: api.SelectedMap("hmac-sha256"), Secret: generated.Secret,
	})
	if err != nil {
		t.Fatalf("AddTsigKey(%q): %v", keyID, err)
	}

	domainID, err = controller.AddPrimaryDomain(ctx, &PrimaryDomain{
		View: api.SelectedMap(viewID), DomainName: domainName, Enabled: "1",
		AllowRndcTransfer: "0", AllowRndcUpdate: "0", UpdateKeys: api.SelectedMapList{keyID},
		UpdatePolicy: api.SelectedMap("zonesub_txt"), DNSSEC: "0",
		TimeToLive: "60", Refresh: "300", Retry: "300", Expire: "86400", Negative: "60",
		MailAdmin: "hostmaster@" + domainName, DnsServer: "ns." + domainName,
	})
	if err != nil {
		t.Fatalf("AddPrimaryDomain(%q): %v", domainID, err)
	}

	for _, record := range []*Record{
		{Domain: api.SelectedMap(domainID), Enabled: "1", Name: "ns", Type: api.SelectedMap("A"), Value: "192.0.2.53"},
		{Domain: api.SelectedMap(domainID), Enabled: "1", Name: "@", Type: api.SelectedMap("NS"), Value: "ns." + domainName + "."},
	} {
		recordID, addErr := controller.AddRecord(ctx, record)
		if recordID != "" {
			recordIDs = append(recordIDs, recordID)
		}
		if addErr != nil {
			t.Fatalf("AddRecord(%q): %v", recordID, addErr)
		}
	}

	view, err := controller.GetView(ctx, viewID)
	if err != nil || view.Name != name || view.MatchClients.String() != aclID {
		t.Fatalf("GetView() = %+v, %v", view, err)
	}
	tsig, err := controller.GetTsigKey(ctx, keyID)
	if err != nil {
		t.Fatalf("GetTsigKey(): %v", err)
	}
	if tsig.Name != name || tsig.Secret == "" {
		t.Fatalf("GetTsigKey() returned unexpected metadata: name=%q secret_present=%t", tsig.Name, tsig.Secret != "")
	}
	domain, err := controller.GetPrimaryDomain(ctx, domainID)
	if err != nil || domain.View.String() != viewID || domain.UpdateKeys.String() != keyID || domain.UpdatePolicy.String() != "zonesub_txt" {
		t.Fatalf("GetPrimaryDomain() = %+v, %v", domain, err)
	}

	if result, err := controller.SearchAcl(ctx); err != nil || result.Total < 1 {
		t.Fatalf("SearchAcl() = %+v, %v", result, err)
	}
	if result, err := controller.SearchView(ctx); err != nil || result.Total < 1 {
		t.Fatalf("SearchView() = %+v, %v", result, err)
	}
	if result, err := controller.SearchTsigKey(ctx); err != nil {
		t.Fatalf("SearchTsigKey(): %v", err)
	} else if result.Total < 1 {
		t.Fatal("SearchTsigKey() returned no rows")
	}
	if result, err := controller.SearchPrimaryDomain(ctx); err != nil || result.Total < 1 {
		t.Fatalf("SearchPrimaryDomain() = %+v, %v", result, err)
	}
	if result, err := controller.SearchRecord(ctx); err != nil || result.Total < 1 {
		t.Fatalf("SearchRecord() = %+v, %v", result, err)
	}
}
