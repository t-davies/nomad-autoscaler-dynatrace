package plugin

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad-autoscaler/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPMPlugin_SetConfig(t *testing.T) {
	cases := []struct {
		name     string
		config   map[string]string
		expected error
	}{
		{
			name:     "no required config parameters are set",
			config:   map[string]string{},
			expected: errors.New("\"DYNATRACE_TENANT_URL\" must not be empty"),
		},
		{
			name: "only tenant_url is set",
			config: map[string]string{
				"tenant_url": "https://abc123456.live.dynatrace.com",
			},
			expected: errors.New("\"DYNATRACE_API_TOKEN\" must not be empty"),
		},
		{
			name: "all required config parameters are set",
			config: map[string]string{
				"tenant_url": "https://abc123456.live.dynatrace.com",
				"api_token":  "dt0s01.a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0u1b2w3x4y5z",
			},
			expected: nil,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			plugin := APMPlugin{logger: hclog.NewNullLogger()}

			err := plugin.SetConfig(c.config)
			assert.Equal(t, c.expected, err, c.name)

			if err == nil {
				// if configured without error, client should be set
				assert.NotNil(t, plugin.client)
			}
		})
	}
}

func TestAPMPlugin_Query(t *testing.T) {
	cases := []struct {
		name      string
		fixture   string
		query     string
		timeRange sdk.TimeRange
		validate  func(*testing.T, sdk.TimestampedMetrics, error)
	}{
		{
			name:    "success (single metric)",
			fixture: "v2_metrics_query_200.json",
			query:   "nomad.client.allocs.cpu.total_percent:filter(eq(task,example-task-name)):avg",
			timeRange: sdk.TimeRange{
				From: time.Unix(1681041120, 0),
				To:   time.Unix(1681041720, 0),
			},
			validate: func(t *testing.T, tm sdk.TimestampedMetrics, err error) {
				assert.NoError(t, err)
				assert.Len(t, tm, 10)

				// validate timestamps parsed correctly (vs. fixture data)
				firstTs := time.Unix(1681041120-1, 0)
				finalTs := time.Unix(1681041720+1, 0)

				for _, tm := range tm {
					assert.Truef(t, tm.Timestamp.After(firstTs),
						"%s is not within the time range, timestamp parsed incorrectly", tm.Timestamp)

					assert.Truef(t, tm.Timestamp.Before(finalTs),
						"%s is not within the time range, timestamp parsed incorrectly", tm.Timestamp)
				}
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			expectedApiToken := "dt0s01.a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0u1b2w3x4y5z"

			stub := httptest.NewServer(
				http.HandlerFunc(
					func(w http.ResponseWriter, r *http.Request) {
						// headers
						assert.Equal(t,
							fmt.Sprintf("Api-Token %s", expectedApiToken),
							r.Header.Get("Authorization"))

						// query params
						v := r.URL.Query()
						assert.Equal(t, c.query, v.Get("metricSelector"))
						assert.Equal(t, "1681041120000", v.Get("from"))
						assert.Equal(t, "1681041720000", v.Get("to"))

						http.ServeFile(w, r, path.Join("./fixtures", c.fixture))
					},
				),
			)
			defer stub.Close()

			plugin := NewDynatracePlugin(hclog.NewNullLogger())
			err := plugin.SetConfig(map[string]string{
				"tenant_url": stub.URL,
				"api_token":  expectedApiToken,
			})

			// if plugin didn't configure, don't bother continuing
			require.NoError(t, err)

			metrics, err := plugin.Query(c.query, c.timeRange)
			c.validate(t, metrics, err)
		})
	}
}
