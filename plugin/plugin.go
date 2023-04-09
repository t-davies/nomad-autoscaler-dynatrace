package plugin

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	hclog "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad-autoscaler/plugins"
	"github.com/hashicorp/nomad-autoscaler/plugins/apm"
	"github.com/hashicorp/nomad-autoscaler/plugins/base"
	"github.com/hashicorp/nomad-autoscaler/sdk"

	"github.com/dynatrace-ace/dynatrace-go-api-client/api/v2/environment/dynatrace"
)

const (
	pluginName = "dynatrace-apm"

	configTenantURLKey = "tenant_url"
	configApiTokenKey  = "api_token"

	envTenantUrlKey = "DYNATRACE_TENANT_URL"
	envApiTokenKey  = "DYNATRACE_API_TOKEN"
)

var (
	_ apm.APM = (*APMPlugin)(nil)

	PluginID = plugins.PluginID{
		Name:       pluginName,
		PluginType: sdk.PluginTypeAPM,
	}

	PluginConfig = &plugins.InternalPluginConfig{
		Factory: func(l hclog.Logger) interface{} { return NewDynatracePlugin(l) },
	}

	pluginInfo = &base.PluginInfo{
		Name:       pluginName,
		PluginType: sdk.PluginTypeAPM,
	}
)

type APMPlugin struct {
	logger    hclog.Logger
	client    *dynatrace.APIClient
	clientCtx context.Context
	config    map[string]string
}

func NewDynatracePlugin(log hclog.Logger) apm.APM {
	return &APMPlugin{
		logger: log,
	}
}

func (p *APMPlugin) SetConfig(config map[string]string) error {
	p.config = config

	tu, err := ValueOrFallback(p.config[configTenantURLKey], FallbackFromEnv(envTenantUrlKey))
	if err != nil {
		return err
	}
	p.config[configTenantURLKey] = tu

	at, err := ValueOrFallback(p.config[configApiTokenKey], FallbackFromEnv(envApiTokenKey))
	if err != nil {
		return err
	}
	p.config[configApiTokenKey] = at

	tenantUrl, err := url.Parse(p.config[configTenantURLKey])
	if err != nil {
		return fmt.Errorf("%q is not a valid URL", configTenantURLKey)
	}

	ctx := context.WithValue(
		context.WithValue(
			context.Background(),
			dynatrace.ContextServerVariables,
			map[string]string{
				"name": tenantUrl.Host,
			},
		),
		dynatrace.ContextAPIKeys,
		map[string]dynatrace.APIKey{
			"Api-Token": {
				Prefix: "Api-Token",
				Key:    p.config[configApiTokenKey],
			},
		},
	)

	p.clientCtx = ctx

	configuration := dynatrace.NewConfiguration()
	configuration.Scheme = tenantUrl.Scheme

	client := dynatrace.NewAPIClient(configuration)
	p.client = client

	return nil
}

func (p *APMPlugin) PluginInfo() (*base.PluginInfo, error) {
	return pluginInfo, nil
}

func (p *APMPlugin) QueryMultiple(q string, r sdk.TimeRange) ([]sdk.TimestampedMetrics, error) {
	ctx, cancel := context.WithTimeout(p.clientCtx, 10*time.Second)
	defer cancel()

	qr, res, err := p.client.MetricsApi.Query(ctx).
		MetricSelector(q).
		From(strconv.FormatInt(r.From.UnixMilli(), 10)).
		To(strconv.FormatInt(r.To.UnixMilli(), 10)).
		Resolution("1m").
		Execute()

	if err != nil {
		if res != nil && res.StatusCode == http.StatusTooManyRequests {
			return nil, fmt.Errorf("metrics query rate limited by dynatrace, wait and retry")
		}

		return nil, fmt.Errorf("error querying metrics from dynatrace: %v", err)
	}

	collections := qr.GetResult()
	if len(collections) == 0 {
		p.logger.Warn("empty metrics response from dynatrace, try a wider query window")
		return nil, nil
	}

	var results []sdk.TimestampedMetrics

	for _, c := range collections {
		series, ok := c.GetDataOk()

		if !ok {
			continue
		}

		var result sdk.TimestampedMetrics

		for _, s := range *series {
			values := s.GetValues()
			for i, ts := range s.GetTimestamps() {
				tm := sdk.TimestampedMetric{
					Timestamp: time.UnixMilli(ts),
					Value:     values[i],
				}

				result = append(result, tm)
			}

			results = append(results, result)
		}
	}

	if len(results) == 0 {
		p.logger.Warn("no data points found in the metrics response from dynatrace, try a wider query window")
	}

	return results, nil
}

func (p *APMPlugin) Query(q string, r sdk.TimeRange) (sdk.TimestampedMetrics, error) {
	m, err := p.QueryMultiple(q, r)
	if err != nil {
		return nil, err
	}

	switch len(m) {
	case 0:
		return sdk.TimestampedMetrics{}, nil
	case 1:
		return m[0], nil
	default:
		return nil, fmt.Errorf("query returned data for %d time series, only 1 is expected", len(m))
	}
}
