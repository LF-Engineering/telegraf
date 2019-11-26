# GitHub Enhanced Input Plugin

Gather Pull Requests, Commits, and Issues information from [GitHub][] hosted repositories.

### Configuration

```toml
[[inputs.github_enhanced]]
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

### Example Output

```
github_repository,language=Go,license=MIT\ License,name=telegraf,owner=influxdata forks=2679i,networks=2679i,open_issues=794i,size=23263i,stars=7091i,subscribers=316i,watchers=7091i 1563901372000000000
internal_github,access_token=Unauthenticated rate_limit_remaining=59i,rate_limit_limit=60i,rate_limit_blocks=0i 1552653551000000000
```
