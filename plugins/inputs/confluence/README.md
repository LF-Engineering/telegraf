# Confluence Input Plugin

Gather data from given confluence url

### Configuration

```toml
[[inputs.confluence]]
  ## The confluence url
  url = "https://confluence.linuxfoundation.org"
  # username = "admin"
  # password = "admin"
    
  ## Set response_timeout
  http_timeout = "5s"
    
  ## Worker pool for confluence plugin only
  ## Empty this field will use default value 5
  # max_connections = 5
```

TODO
