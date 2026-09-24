package cloudflare

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strconv"
	"sync"
	"testing"
)

// Values reused across this package's test files, factored out so
// goconst doesn't flag the same literal scattered across files.
const (
	testAPIToken  = "tok"
	testHostWWW   = "www"
	testAddrA1    = "1.1.1.1"
	testAddrA2    = "2.2.2.2"
	testSPFValue  = "v=spf1 include:example.com ~all"
	testRecordID1 = "rec-1"
)

// fakeServer is an in-memory httptest-backed stand-in for the Cloudflare
// API v4. It never touches the network beyond the loopback httptest
// listener, and models just enough of the real API - pagination, the
// success/error envelope, per-zone DNS records, and the
// ZoneConfigurer-related sub-resources - to exercise Provider without a
// live Cloudflare account.
type fakeServer struct {
	t *testing.T

	mu        sync.Mutex
	zones     map[string]zoneWire
	records   map[string]map[string]dnsRecordWire // zoneID -> recordID -> record
	settings  map[string]map[string]interface{}   // zoneID -> settingID -> value
	botMgmt   map[string]map[string]interface{}   // zoneID -> raw bot_management object
	secTXT    map[string]securityTXTWire
	redirects map[string]rulesetWire
	dnssec    map[string]dnssecWire
	nextID    int
	calls     []string // "METHOD /path" in request order, for call-count assertions

	*httptest.Server
}

func newFakeServer(t *testing.T) *fakeServer {
	t.Helper()
	fs := &fakeServer{
		t:         t,
		zones:     map[string]zoneWire{},
		records:   map[string]map[string]dnsRecordWire{},
		settings:  map[string]map[string]interface{}{},
		botMgmt:   map[string]map[string]interface{}{},
		secTXT:    map[string]securityTXTWire{},
		redirects: map[string]rulesetWire{},
		dnssec:    map[string]dnssecWire{},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /zones", fs.handleListZones)
	mux.HandleFunc("GET /zones/{zoneID}", fs.handleGetZone)
	mux.HandleFunc("GET /zones/{zoneID}/dns_records", fs.handleListRecords)
	mux.HandleFunc("POST /zones/{zoneID}/dns_records", fs.handleCreateRecord)
	mux.HandleFunc("PUT /zones/{zoneID}/dns_records/{recordID}", fs.handleUpdateRecord)
	mux.HandleFunc("DELETE /zones/{zoneID}/dns_records/{recordID}", fs.handleDeleteRecord)
	mux.HandleFunc("GET /zones/{zoneID}/settings", fs.handleGetSettings)
	mux.HandleFunc("PATCH /zones/{zoneID}/settings", fs.handlePatchSettings)
	mux.HandleFunc("GET /zones/{zoneID}/bot_management", fs.handleGetBotManagement)
	mux.HandleFunc("PUT /zones/{zoneID}/bot_management", fs.handlePutBotManagement)
	mux.HandleFunc("GET /zones/{zoneID}/security-center/securitytxt", fs.handleGetSecurityTXT)
	mux.HandleFunc("PUT /zones/{zoneID}/security-center/securitytxt", fs.handlePutSecurityTXT)
	mux.HandleFunc("GET /zones/{zoneID}/rulesets/phases/http_request_dynamic_redirect/entrypoint", fs.handleGetRuleset)
	mux.HandleFunc("PUT /zones/{zoneID}/rulesets/phases/http_request_dynamic_redirect/entrypoint", fs.handlePutRuleset)
	mux.HandleFunc("GET /zones/{zoneID}/dnssec", fs.handleGetDNSSEC)
	mux.HandleFunc("PATCH /zones/{zoneID}/dnssec", fs.handlePatchDNSSEC)

	fs.Server = httptest.NewServer(fs.logCalls(mux))
	t.Cleanup(fs.Close)
	return fs
}

func (fs *fakeServer) logCalls(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fs.mu.Lock()
		fs.calls = append(fs.calls, r.Method+" "+r.URL.Path)
		fs.mu.Unlock()
		next.ServeHTTP(w, r)
	})
}

func (fs *fakeServer) callCount(method, path string) int {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	n := 0
	for _, c := range fs.calls {
		if c == method+" "+path {
			n++
		}
	}
	return n
}

func (fs *fakeServer) addZone(id, name string) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	fs.zones[id] = zoneWire{ID: id, Name: name}
	if fs.records[id] == nil {
		fs.records[id] = map[string]dnsRecordWire{}
	}
}

func (fs *fakeServer) seedRecord(zoneID string, w dnsRecordWire) dnsRecordWire {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	if w.ID == "" {
		fs.nextID++
		w.ID = fmt.Sprintf("rec-%d", fs.nextID)
	}
	fs.records[zoneID][w.ID] = w
	return w
}

// --- response helpers ---

func writeSuccess(w http.ResponseWriter, result interface{}, info *resultInfo) {
	env := apiEnvelope{Success: true}
	if result != nil {
		raw, err := json.Marshal(result)
		if err != nil {
			panic(err)
		}
		env.Result = raw
	}
	env.ResultInfo = info
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(env)
}

func writeError(w http.ResponseWriter, status, code int, message string) {
	env := apiEnvelope{Success: false, Errors: []apiError{{Code: code, Message: message}}}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(env)
}

// --- zones ---

func (fs *fakeServer) handleListZones(w http.ResponseWriter, r *http.Request) {
	fs.mu.Lock()
	name := r.URL.Query().Get("name")
	var all []zoneWire
	for _, z := range fs.zones {
		if name == "" || z.Name == name {
			all = append(all, z)
		}
	}
	fs.mu.Unlock()

	sort.Slice(all, func(i, j int) bool { return all[i].ID < all[j].ID })

	page, info := slicePage(all, r)
	writeSuccess(w, page, info)
}

func (fs *fakeServer) handleGetZone(w http.ResponseWriter, r *http.Request) {
	fs.mu.Lock()
	z, ok := fs.zones[r.PathValue("zoneID")]
	fs.mu.Unlock()
	if !ok {
		writeError(w, http.StatusNotFound, 1001, "zone not found")
		return
	}
	writeSuccess(w, z, nil)
}

// --- dns records ---

func (fs *fakeServer) handleListRecords(w http.ResponseWriter, r *http.Request) {
	zoneID := r.PathValue("zoneID")

	fs.mu.Lock()
	all := make([]dnsRecordWire, 0, len(fs.records[zoneID]))
	for _, rec := range fs.records[zoneID] {
		all = append(all, rec)
	}
	fs.mu.Unlock()

	sort.Slice(all, func(i, j int) bool { return all[i].ID < all[j].ID })

	page, info := slicePage(all, r)
	writeSuccess(w, page, info)
}

// slicePage applies Cloudflare-style page/per_page query params to an
// already-deterministically-ordered slice and builds the matching
// result_info envelope. Callers must sort `all` first: two separate
// requests (page 1, then page 2) must see a stable order, which a bare
// map iteration does not guarantee.
func slicePage[T any](all []T, r *http.Request) ([]T, *resultInfo) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if perPage < 1 {
		perPage = listPerPage
	}

	totalPages := (len(all) + perPage - 1) / perPage
	if totalPages == 0 {
		totalPages = 1
	}

	start := (page - 1) * perPage
	if start > len(all) {
		start = len(all)
	}
	end := start + perPage
	if end > len(all) {
		end = len(all)
	}

	return all[start:end], &resultInfo{
		Page: page, PerPage: perPage, TotalPages: totalPages,
		Count: end - start, TotalCount: len(all),
	}
}

func (fs *fakeServer) handleCreateRecord(w http.ResponseWriter, r *http.Request) {
	zoneID := r.PathValue("zoneID")
	var rec dnsRecordWire
	if !decodeBody(fs.t, w, r, &rec) {
		return
	}

	fs.mu.Lock()
	fs.nextID++
	rec.ID = fmt.Sprintf("rec-%d", fs.nextID)
	if fs.records[zoneID] == nil {
		fs.records[zoneID] = map[string]dnsRecordWire{}
	}
	fs.records[zoneID][rec.ID] = rec
	fs.mu.Unlock()

	writeSuccess(w, rec, nil)
}

func (fs *fakeServer) handleUpdateRecord(w http.ResponseWriter, r *http.Request) {
	zoneID, recordID := r.PathValue("zoneID"), r.PathValue("recordID")

	var rec dnsRecordWire
	if !decodeBody(fs.t, w, r, &rec) {
		return
	}

	fs.mu.Lock()
	if _, ok := fs.records[zoneID][recordID]; !ok {
		fs.mu.Unlock()
		writeError(w, http.StatusNotFound, 81044, "record not found")
		return
	}
	rec.ID = recordID
	fs.records[zoneID][recordID] = rec
	fs.mu.Unlock()

	writeSuccess(w, rec, nil)
}

func (fs *fakeServer) handleDeleteRecord(w http.ResponseWriter, r *http.Request) {
	zoneID, recordID := r.PathValue("zoneID"), r.PathValue("recordID")

	fs.mu.Lock()
	if _, ok := fs.records[zoneID][recordID]; !ok {
		fs.mu.Unlock()
		writeError(w, http.StatusNotFound, 81044, "record not found")
		return
	}
	delete(fs.records[zoneID], recordID)
	fs.mu.Unlock()

	writeSuccess(w, map[string]string{"id": recordID}, nil)
}

// --- zone settings ---

func (fs *fakeServer) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	zoneID := r.PathValue("zoneID")
	fs.mu.Lock()
	settings := fs.settings[zoneID]
	fs.mu.Unlock()

	out := make([]zoneSettingWire, 0, len(managedZoneSettings))
	for _, id := range managedZoneSettings {
		if v, ok := settings[id]; ok {
			out = append(out, zoneSettingWire{ID: id, Value: v})
		}
	}
	// Include one unmanaged setting to prove GetZoneSettings filters it out.
	out = append(out, zoneSettingWire{ID: "brotli", Value: "on"})
	writeSuccess(w, out, nil)
}

func (fs *fakeServer) handlePatchSettings(w http.ResponseWriter, r *http.Request) {
	zoneID := r.PathValue("zoneID")
	var body struct {
		Items []zoneSettingWire `json:"items"`
	}
	if !decodeBody(fs.t, w, r, &body) {
		return
	}

	fs.mu.Lock()
	if fs.settings[zoneID] == nil {
		fs.settings[zoneID] = map[string]interface{}{}
	}
	for _, item := range body.Items {
		fs.settings[zoneID][item.ID] = item.Value
	}
	fs.mu.Unlock()

	writeSuccess(w, body.Items, nil)
}

// --- bot management ---

func (fs *fakeServer) handleGetBotManagement(w http.ResponseWriter, r *http.Request) {
	zoneID := r.PathValue("zoneID")
	fs.mu.Lock()
	raw := fs.botMgmt[zoneID]
	fs.mu.Unlock()
	writeSuccess(w, raw, nil)
}

func (fs *fakeServer) handlePutBotManagement(w http.ResponseWriter, r *http.Request) {
	zoneID := r.PathValue("zoneID")
	var raw map[string]interface{}
	if !decodeBody(fs.t, w, r, &raw) {
		return
	}
	fs.mu.Lock()
	fs.botMgmt[zoneID] = raw
	fs.mu.Unlock()
	writeSuccess(w, raw, nil)
}

// --- security.txt ---

func (fs *fakeServer) handleGetSecurityTXT(w http.ResponseWriter, r *http.Request) {
	zoneID := r.PathValue("zoneID")
	fs.mu.Lock()
	txt := fs.secTXT[zoneID]
	fs.mu.Unlock()
	writeSuccess(w, txt, nil)
}

func (fs *fakeServer) handlePutSecurityTXT(w http.ResponseWriter, r *http.Request) {
	zoneID := r.PathValue("zoneID")
	var txt securityTXTWire
	if !decodeBody(fs.t, w, r, &txt) {
		return
	}
	fs.mu.Lock()
	fs.secTXT[zoneID] = txt
	fs.mu.Unlock()
	writeSuccess(w, txt, nil)
}

// --- redirect ruleset ---

func (fs *fakeServer) handleGetRuleset(w http.ResponseWriter, r *http.Request) {
	zoneID := r.PathValue("zoneID")
	fs.mu.Lock()
	rs := fs.redirects[zoneID]
	fs.mu.Unlock()
	writeSuccess(w, rs, nil)
}

func (fs *fakeServer) handlePutRuleset(w http.ResponseWriter, r *http.Request) {
	zoneID := r.PathValue("zoneID")
	var rs rulesetWire
	if !decodeBody(fs.t, w, r, &rs) {
		return
	}
	fs.mu.Lock()
	fs.redirects[zoneID] = rs
	fs.mu.Unlock()
	writeSuccess(w, rs, nil)
}

// --- dnssec ---

func (fs *fakeServer) handleGetDNSSEC(w http.ResponseWriter, r *http.Request) {
	zoneID := r.PathValue("zoneID")
	fs.mu.Lock()
	d := fs.dnssec[zoneID]
	fs.mu.Unlock()
	writeSuccess(w, d, nil)
}

func (fs *fakeServer) handlePatchDNSSEC(w http.ResponseWriter, r *http.Request) {
	zoneID := r.PathValue("zoneID")
	var body struct {
		Status string `json:"status"`
	}
	if !decodeBody(fs.t, w, r, &body) {
		return
	}

	fs.mu.Lock()
	d := fs.dnssec[zoneID]
	d.Status = body.Status
	if body.Status == "active" && d.DS == "" {
		d.DS = "example.com. 3600 IN DS 2371 13 2 (fake-digest)"
	}
	fs.dnssec[zoneID] = d
	fs.mu.Unlock()

	writeSuccess(w, d, nil)
}

func decodeBody(t *testing.T, w http.ResponseWriter, r *http.Request, out interface{}) bool {
	t.Helper()
	if err := json.NewDecoder(r.Body).Decode(out); err != nil {
		writeError(w, http.StatusBadRequest, 1000, "invalid JSON: "+err.Error())
		return false
	}
	return true
}

func newTestProvider(t *testing.T, fs *fakeServer) *Provider {
	t.Helper()
	p, err := New(Config{
		APIToken:  "test-token",
		AccountID: "test-account",
		BaseURL:   fs.URL,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return p
}
