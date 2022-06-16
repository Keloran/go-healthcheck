package healthcheck_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/keloran/go-healthcheck"
	"github.com/stretchr/testify/assert"
)

func TestHealthCheck_Check(t *testing.T) {
	tests := []struct {
		request healthcheck.HealthCheck
		expect  healthcheck.Health
		err     error
	}{
		{
			request: healthcheck.HealthCheck{
				Name:         "test1",
				URL:          "test1.com",
				Dependencies: "",
			},
			expect: healthcheck.Health{
				Name:         "test1",
				URL:          "test1.com",
				Status:       healthcheck.HealthPass,
				Dependencies: nil,
			},
			err: nil,
		},
		{
			request: healthcheck.HealthCheck{
				Name:         "test2",
				URL:          "test1.com",
				Dependencies: fmt.Sprintf(`{"dependencies":[{"name":"%s","url":"%s","ping":true}]}`, "test1", "test1.com"),
			},
			expect: healthcheck.Health{
				Name:   "test2",
				URL:    "test1.com",
				Status: healthcheck.HealthPass,
				Dependencies: []healthcheck.Health{
					{
						Name:         "test1",
						URL:          "test1.com",
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

func TestHTTP(t *testing.T) {
	tests := []struct {
		request healthcheck.HealthCheck
		expect  healthcheck.Health
	}{
		{
			request: healthcheck.HealthCheck{
				Name: "test1",
				URL:  "test1.com",
			},
			expect: healthcheck.Health{
				Name:         "test1",
				Dependencies: nil,
				Status:       healthcheck.HealthPass,
			},
		},
	}

	for _, test := range tests {
		os.Setenv("SERVICE_NAME", test.request.Name)
		jsonRequest, _ := json.Marshal(test.request)
		request, _ := http.NewRequest("GET", "/", bytes.NewBuffer(jsonRequest))
		response := httptest.NewRecorder()
		healthcheck.HTTP(response, request)
		assert.Equal(t, 200, response.Code)

		body, _ := ioutil.ReadAll(response.Body)
		healthy := healthcheck.Health{}
		_ = json.Unmarshal(body, &healthy)
		assert.Equal(t, test.expect, healthy)
	}
}
