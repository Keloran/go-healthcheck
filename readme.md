# Healthcheck

This allows you to run health checks for the system and dependencies

### Guide
You need to put the following json into your SERVICE_DEPENDENCIES environment setting

```
{
  "dependencies":[{
    "name": <service_name>,
    "url": <service_url>,
    "ping": <is it just a ping test or a full healthcheck>,
    "type": <optional: "postgres" or "valkey"/"redis">
  }]
}
```

With `type` set, `url` can be `host`, `host:port` or a DSN (`postgres://user:pass@host:5432/db`, `$RDS_URL` from keloran/go-config).
The check talks the wire protocol directly (no driver, no credentials needed), and only `host:port` is reported back.

#### Optional Settings
SERVICE_NAME // sets the name otherwise it is blank