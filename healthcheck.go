package healthcheck

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	probing "github.com/prometheus-community/pro-bing"
)

// HTTP the request as done by routing
func HTTP(w http.ResponseWriter, r *http.Request) {
	hc := HealthCheck{
		Name:         os.Getenv("SERVICE_NAME"),
		URL:          r.Host,
		Dependencies: os.Getenv("SERVICE_DEPENDENCIES"),
	}

	health, err := hc.Check()
	if err != nil {
		w.Header().Set("Content-Type", "application/health+json")
		j, _ := json.Marshal(Health{
			Status: HealthFail,
		})
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(j); err != nil {
			fmt.Printf("write response: %v\n", err)
		}
		fmt.Printf("http health failed: %+v\n", err)
		return
	}

	j, _ := json.Marshal(health)
	w.Header().Set("Content-Type", "application/health+json")
	if _, err := w.Write(j); err != nil {
		fmt.Printf("write response: %v\n", err)
	}
}

// Check do the health check itself
func (h HealthCheck) Check() (Health, error) {
	health := Health{
		Name:         h.Name,
		URL:          h.URL,
		Status:       HealthFail,
		Dependencies: nil,
	}

	if h.Dependencies != "" {
		deps, err := h.getDependencies()
		if err != nil {
			return health, err
		}

		checkedDeps := []Health{}
		for _, dep := range deps.Dependencies {
			d, err := dep.check()
			if err != nil {
				return health, err
			}
			checkedDeps = append(checkedDeps, d)
		}

		health.Dependencies = checkedDeps
	}

	// now set to failed if a dependency failed
	health.Status = HealthPass
	for _, dep := range health.Dependencies {
		if dep.Status == HealthFail {
			health.Status = HealthFail
		}
	}

	return health, nil
}

// getDependencies get the list of dependencies
func (h HealthCheck) getDependencies() (Dependencies, error) {
	deps := Dependencies{}
	if err := json.Unmarshal([]byte(h.Dependencies), &deps); err != nil {
		return deps, err
	}

	return deps, nil
}

// check the dependency status
func (d Dependency) check() (Health, error) {
	if strings.Contains(d.URL, "$") {
		d.URL = os.Getenv(d.URL[1:])
	}

	switch d.Type {
	case "postgres":
		// SSLRequest packet: any server answers 'S' or 'N' before auth, so no creds needed
		return d.probe("5432", []byte{0, 0, 0, 8, 4, 210, 22, 47}, func(b []byte) bool {
			return len(b) == 1 && (b[0] == 'S' || b[0] == 'N')
		})
	case "valkey", "redis":
		// NOAUTH still means the server is up, we just can't run commands without a password
		return d.probe("6379", []byte("PING\r\n"), func(b []byte) bool {
			return strings.HasPrefix(string(b), "+PONG") || strings.HasPrefix(string(b), "-NOAUTH")
		})
	}

	// Ping check
	if d.Ping {
		return d.ping()
	}

	// Standard check
	return d.curl()
}

// ping checks
func (d Dependency) ping() (Health, error) {
	h := Health{
		Name:   d.Name,
		URL:    d.URL,
		Status: HealthFail,
	}

	pinger, err := probing.NewPinger(h.URL)
	if err != nil {
		return h, err
	}

	pinger.Count = 3
	if err := pinger.Run(); err != nil {
		return h, err
	}

	h.Status = HealthPass
	return h, nil
}

// curl checks
func (d Dependency) curl() (Health, error) {
	h := Health{}
	p, err := http.Get(d.URL)
	if err != nil {
		h = Health{
			URL:    d.URL,
			Status: HealthFail,
		}
		return h, err
	}

	defer func() {
		_ = p.Body.Close()
	}()

	b, err := io.ReadAll(p.Body)
	if err != nil {
		h = Health{
			URL:    d.URL,
			Status: HealthFail,
		}
		return h, err
	}
	if err := json.Unmarshal(b, &h); err != nil {
		return h, err
	}
	return h, nil
}

// probe dials the dependency, sends req and checks the reply
// ponytail: plain TCP only, add tls.Dial when a dep needs sslmode=require / rediss
func (d Dependency) probe(defaultPort string, req []byte, ok func([]byte) bool) (Health, error) {
	// accepts host, host:port or a full DSN like postgres://user:pass@host:5432/db
	addr := d.URL
	if u, err := url.Parse(d.URL); err == nil && u.Host != "" {
		addr = u.Host
	}
	if _, _, err := net.SplitHostPort(addr); err != nil {
		addr = net.JoinHostPort(addr, defaultPort)
	}

	// report host:port only, never the DSN (it can carry a password)
	h := Health{
		Name:   d.Name,
		URL:    addr,
		Status: HealthFail,
	}

	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return h, err
	}
	defer func() {
		_ = conn.Close()
	}()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))

	if _, err := conn.Write(req); err != nil {
		return h, err
	}
	b := make([]byte, 64)
	n, err := conn.Read(b)
	if err != nil {
		return h, err
	}
	if !ok(b[:n]) {
		return h, fmt.Errorf("unexpected response from %s: %q", addr, b[:n])
	}

	h.Status = HealthPass
	return h, nil
}
