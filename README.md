# Dynatrace APM Plugin

[![ci](https://github.com/t-davies/nomad-autoscaler-dynatrace/actions/workflows/ci.yml/badge.svg)](https://github.com/t-davies/nomad-autoscaler-dynatrace/actions/workflows/ci.yml) [![release](https://img.shields.io/github/v/release/t-davies/nomad-autoscaler-dynatrace?include_prereleases&sort=semver&logo=github)](https://github.com/t-davies/nomad-autoscaler-dynatrace/releases)

The `dynatrace-apm` APM plugin allows querying for metrics from [Dynatrace](https://www.dynatrace.com/).

## Agent Configuration Options

```hcl
apm "dynatrace" {
  driver = "dynatrace-apm"

  config = {
    tenant_url = "https://{your-environment-id}.live.dynatrace.com"
    api_token  = "dt0s01.XXXXXXXXXXXXXXX"
  }
}
```

- `tenant_url` `(string: <required>)` - the URL of the Dynatrace tenant to query
- `api_token` `(string: <required>)` - the [access token](https://www.dynatrace.com/support/help/dynatrace-api/basics/dynatrace-api-authentication) to use to authenticate with Dynatrace

The Dynatrace plugin can also read its configuration options from environment variables. The accepted keys are `DYNATRACE_TENANT_URL` and `DYNATRACE_API_TOKEN`. The agent configuration parameters take precedence over the environment variables.

## Policy Configuration Options

```hcl
check {
  source = "dynatrace"
  query  = "nomad.client.allocs.cpu.total_percent:filter(eq(task,example-task-name)):avg"
}
```
