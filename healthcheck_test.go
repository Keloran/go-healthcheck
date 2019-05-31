package healthcheck_test

import (
	"bytes"
	"encoding/json"
	"github.com/keloran/go-healthcheck"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
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
				URL:          "test1.com",
				Dependencies: nil,
				Status:       healthcheck.HealthPass,
			},
		},
	}

	for _, test := range tests {
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
