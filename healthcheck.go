package healthcheck

import (
  "encoding/json"
  "fmt"
  "io"
  "net/http"
  "os"
  "strings"

  "github.com/go-ping/ping"
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

  health.Status = HealthPass
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

  pinger, err := ping.NewPinger(h.URL)
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
