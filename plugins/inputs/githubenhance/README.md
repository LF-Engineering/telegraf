# GitHub Enhanced Input Plugin

Gather Pull Requests, Commits, and Issues information from [GitHub][] hosted repositories.

### Configuration

```toml
[[inputs.githubenhanced]]
  ## List of repositories to monitor
  repositories = [
	  "influxdata/telegraf",
	  "influxdata/influxdb"
  ]

  ## Github API access token.  Unauthenticated requests are limited to 60 per hour.
  # access_token = ""
```

### Metrics
TODO

