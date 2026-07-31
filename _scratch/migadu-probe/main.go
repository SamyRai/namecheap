// Command migadu-probe establishes what Migadu's Admin API actually accepts,
// as opposed to what its documentation describes.
//
// It is read-only unless -write is passed, and it always reads state back after
// a write rather than trusting the response status: this API has been observed
// returning HTTP 200 while silently coercing the value it was given.
//
//	go run ./_scratch/migadu-probe/main.go -domain example.com
//	go run ./_scratch/migadu-probe/main.go -domain example.com -probe-alias -write
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

const baseURL = "https://api.migadu.com/v1"

type probe struct {
	account string
	key     string
	client  *http.Client
	write   bool
}

func main() {
	var (
		domain     = flag.String("domain", "", "domain to probe (required)")
		probeAlias = flag.Bool("probe-alias", false, "test whether domain-level aliasing is settable via the API")
		create     = flag.Bool("create", false, "create the domain on the account first (requires -write)")
		defaultsOn = flag.Bool("default-addresses", false, "with -create, ask Migadu to create its default addresses")
		write      = flag.Bool("write", false, "permit PATCH/POST requests (default: read-only)")
	)
	flag.Parse()

	if *domain == "" {
		fmt.Fprintln(os.Stderr, "-domain is required")
		os.Exit(2)
	}
	account, key := os.Getenv("MIGADU_ACCOUNT"), os.Getenv("MIGADU_API_KEY")
	if account == "" || key == "" {
		fmt.Fprintln(os.Stderr, "MIGADU_ACCOUNT and MIGADU_API_KEY must be set")
		fmt.Fprintln(os.Stderr, "note: keys stored with a 'label=' prefix must have the prefix stripped")
		os.Exit(2)
	}

	p := &probe{
		account: account,
		key:     key,
		client:  &http.Client{Timeout: 30 * time.Second},
		write:   *write,
	}

	if *create {
		p.createDomain(*domain, *defaultsOn)
	}
	p.dumpDomainFields(*domain)
	if *probeAlias {
		p.probeDomainAliasing(*domain)
	}
}

// createDomain adds a domain to the account. The documented payload is just
// name / create_default_addresses / hosted_dns; this reports what actually
// comes back so the difference is visible.
func (p *probe) createDomain(domain string, defaultAddresses bool) {
	if !p.write {
		fmt.Printf("== create %s: skipped, -write not set ==\n", domain)
		return
	}
	body := map[string]interface{}{
		"name":                     domain,
		"create_default_addresses": defaultAddresses,
		// hosted_dns=false: DNS stays at the registrar, which is the whole
		// point of publishing records through zonekit.
		"hosted_dns": false,
	}
	status, raw, err := p.do(http.MethodPost, "/domains", body)
	if err != nil {
		fmt.Printf("== create %s: ERROR %v ==\n", domain, err)
		return
	}
	fmt.Printf("== POST /domains (%s) -> HTTP %d ==\n", domain, status)
	if status >= 400 {
		fmt.Printf("   %s\n", truncate(string(raw), 300))
	}
}

func (p *probe) do(method, path string, body interface{}) (int, []byte, error) {
	var reader io.Reader = bytes.NewReader(nil)
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return 0, nil, err
		}
		reader = bytes.NewReader(encoded)
	}
	req, err := http.NewRequest(method, baseURL+path, reader)
	if err != nil {
		return 0, nil, err
	}
	req.SetBasicAuth(p.account, p.key)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(resp.Body)
	return resp.StatusCode, raw, err
}

// dumpDomainFields lists every key the domain object actually carries, so a
// field the documentation omits still shows up.
func (p *probe) dumpDomainFields(domain string) {
	status, raw, err := p.do(http.MethodGet, "/domains/"+domain, nil)
	if err != nil {
		fmt.Printf("GET /domains/%s failed: %v\n", domain, err)
		return
	}
	fmt.Printf("== GET /domains/%s -> HTTP %d ==\n", domain, status)
	if status != http.StatusOK {
		fmt.Printf("   %s\n", truncate(string(raw), 300))
		return
	}

	var obj map[string]interface{}
	if err := json.Unmarshal(raw, &obj); err != nil {
		fmt.Printf("   unparseable: %v\n", err)
		return
	}
	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	fmt.Printf("   %d fields:\n", len(keys))
	for _, k := range keys {
		marker := ""
		// Flag anything that looks like it could express domain aliasing.
		for _, hint := range []string{"alias", "primary", "parent", "target", "forward"} {
			if strings.Contains(strings.ToLower(k), hint) {
				marker = "   <-- possible aliasing field"
			}
		}
		fmt.Printf("     %-34s %v%s\n", k, obj[k], marker)
	}
}

// probeDomainAliasing tests whether a domain can be pointed at another domain
// through the API. Migadu's guides describe domain aliasing as a product
// feature, but the documented API surface has no field for it -- so the
// question is whether the field simply went undocumented.
//
// Each candidate is sent alone and the domain is read back afterwards, because
// this API returns 200 for values it silently discards.
func (p *probe) probeDomainAliasing(domain string) {
	candidates := []struct {
		field string
		value interface{}
	}{
		{"domain_alias", "glpx.pro"},
		{"alias_of", "glpx.pro"},
		{"primary_domain", "glpx.pro"},
		{"parent_domain", "glpx.pro"},
		{"is_alias", true},
		{"aliased_to", "glpx.pro"},
		{"forward_to_domain", "glpx.pro"},
	}

	fmt.Printf("\n== probing domain-level aliasing on %s (write=%v) ==\n", domain, p.write)
	if !p.write {
		fmt.Println("   read-only: pass -write to actually attempt the PATCHes")
		return
	}

	before := p.domainSnapshot(domain)

	for _, c := range candidates {
		status, raw, err := p.do(http.MethodPatch, "/domains/"+domain,
			map[string]interface{}{c.field: c.value})
		if err != nil {
			fmt.Printf("   %-20s ERROR %v\n", c.field, err)
			continue
		}

		after := p.domainSnapshot(domain)
		verdict := "ignored (field absent after write)"
		switch {
		case status >= 400:
			verdict = "rejected: " + truncate(string(raw), 90)
		case !equalSnapshots(before, after):
			verdict = "ACCEPTED -- domain object changed"
		}
		fmt.Printf("   %-20s HTTP %-3d %s\n", c.field, status, verdict)
		before = after
	}

	fmt.Println("\n   A field is only real if the object changed. HTTP 200 alone means")
	fmt.Println("   nothing here: this API 200s on values it quietly drops.")
}

func (p *probe) domainSnapshot(domain string) map[string]interface{} {
	_, raw, err := p.do(http.MethodGet, "/domains/"+domain, nil)
	if err != nil {
		return nil
	}
	var obj map[string]interface{}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil
	}
	return obj
}

func equalSnapshots(a, b map[string]interface{}) bool {
	ja, _ := json.Marshal(a)
	jb, _ := json.Marshal(b)
	return string(ja) == string(jb)
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\n", " "))
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
