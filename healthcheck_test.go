package healthcheck_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/keloran/go-healthcheck"
	"github.com/stretchr/testify/assert"
)

func TestHealthCheck_Check(t *testing.T) {
	dep := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"name":"test1","url":"dep","status":"pass"}`))
	}))
	defer dep.Close()

	tests := []struct {
		request healthcheck.HealthCheck
		expect  healthcheck.Health
		err     error
	}{
		{
			request: healthcheck.HealthCheck{
				Name:         "test1",
				URL:          "chewedfeed.com",
				Dependencies: "",
			},
			expect: healthcheck.Health{
				Name:         "test1",
				URL:          "chewedfeed.com",
				Status:       healthcheck.HealthPass,
				Dependencies: nil,
			},
			err: nil,
		},
		{
			request: healthcheck.HealthCheck{
				Name:         "test2",
				URL:          "chewedfeed.com",
				Dependencies: fmt.Sprintf(`{"dependencies":[{"name":"test1","url":"%s"}]}`, dep.URL),
			},
			expect: healthcheck.Health{
				Name:   "test2",
				URL:    "chewedfeed.com",
				Status: healthcheck.HealthPass,
				Dependencies: []healthcheck.Health{
					{
						Name:         "test1",
						URL:          "dep",
						Status:       "pass",
						Dependencies: nil,
					},
				},
			},
		},
	}

	for _, test := range tests {
		response, err := test.request.Check()
		assert.Equal(t, test.err, err)

		assert.Equal(t, test.expect, response)
	}
}

func TestHealthCheck_Ping(t *testing.T) {
	hc := healthcheck.HealthCheck{
		Name:         "svc",
		Dependencies: `{"dependencies":[{"name":"local","url":"127.0.0.1","ping":true}]}`,
	}
	h, err := hc.Check()
	if errors.Is(err, os.ErrPermission) {
		t.Skip("ICMP sockets not allowed here (GitHub runners): ", err)
	}
	assert.NoError(t, err)
	assert.Equal(t, healthcheck.HealthPass, h.Status)
}

func TestHTTP(t *testing.T) {
	tests := []struct {
		request healthcheck.HealthCheck
		expect  healthcheck.Health
	}{
		{
			request: healthcheck.HealthCheck{
				Name: "test1",
				URL:  "chewedfeed.com",
			},
			expect: healthcheck.Health{
				Name:         "test1",
				Dependencies: nil,
				Status:       healthcheck.HealthPass,
			},
		},
	}

	for _, test := range tests {
		t.Setenv("SERVICE_NAME", test.request.Name)
		jsonRequest, _ := json.Marshal(test.request)
		request, _ := http.NewRequest("GET", "/", bytes.NewBuffer(jsonRequest))
		response := httptest.NewRecorder()
		healthcheck.HTTP(response, request)
		assert.Equal(t, 200, response.Code)

		body, _ := io.ReadAll(response.Body)
		healthy := healthcheck.Health{}
		_ = json.Unmarshal(body, &healthy)
		assert.Equal(t, test.expect, healthy)
	}
}

// fakeServer answers every connection with reply, returns its address
func fakeServer(t *testing.T, reply string) string {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = l.Close() })
	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				return
			}
			_, _ = c.Read(make([]byte, 64))
			_, _ = c.Write([]byte(reply))
			_ = c.Close()
		}
	}()
	return l.Addr().String()
}

func TestHealthCheck_DatabaseDeps(t *testing.T) {
	t.Setenv("RDS_URL", "postgres://u:secret@"+fakeServer(t, "S")+"/db")
	tests := []struct {
		name, kind, url string
		expect          healthcheck.HealthStatus
		wantErr         bool
	}{
		{"pg ssl", "postgres", fakeServer(t, "S"), healthcheck.HealthPass, false},
		{"pg nossl dsn", "postgres", "postgres://u:secret@" + fakeServer(t, "N") + "/db", healthcheck.HealthPass, false},
		{"valkey", "valkey", fakeServer(t, "+PONG\r\n"), healthcheck.HealthPass, false},
		{"valkey auth", "redis", fakeServer(t, "-NOAUTH Authentication required.\r\n"), healthcheck.HealthPass, false},
		{"valkey wrong server", "valkey", fakeServer(t, "HTTP/1.1 400"), healthcheck.HealthFail, true},
		{"go-config env", "postgres", "$RDS_URL", healthcheck.HealthPass, false},
		{"down", "postgres", "127.0.0.1:1", healthcheck.HealthFail, true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			hc := healthcheck.HealthCheck{
				Name:         "svc",
				Dependencies: fmt.Sprintf(`{"dependencies":[{"name":"db","url":%q,"type":%q}]}`, test.url, test.kind),
			}
			h, err := hc.Check()
			if test.wantErr {
				assert.Error(t, err)
				assert.Equal(t, healthcheck.HealthFail, h.Status)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, test.expect, h.Status)
			assert.NotContains(t, h.Dependencies[0].URL, "secret")
		})
	}
}
