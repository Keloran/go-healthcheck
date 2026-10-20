package healthcheck

// Health give the status and its dependencies
type Health struct {
	Name         string       `json:"name"`
	URL          string       `json:"url"`
	Status       HealthStatus `json:"status"`
	Dependencies []Health     `json:"dependencies,omitempty"`
}

// Dependencies list the dependencies to test
type Dependencies struct {
	Dependencies []Dependency `json:"dependencies"`
}

// HealthCheck model
type HealthCheck struct {
	URL          string
	Name         string
	Dependencies string
}

// Dependency what to test
type Dependency struct {
	URL  string `json:"url"`
	Name string `json:"name"`
	Ping bool   `json:"ping"`
	// Type is "postgres" or "valkey" (alias "redis") for a protocol-level check, empty for ping/http
	Type string `json:"type"`
}

// HealthStatus define its type
type HealthStatus string

const (
	// HealthPass set its status to passed
	HealthPass HealthStatus = "pass"

	// HealthFail set its status to failed
	HealthFail HealthStatus = "fail"
)
