# Healthcheck

This allows you to run health checks for the system and dependencies

### Guide
You need to put the following json into your SERVICE_DEPENDENCIES environment setting

```
{
  "dependencies":[{
    "name": <service_name>,
    "url": <service_url>,
    "ping": <is it just a ping test or a full healthcheck>
  }]
}
```

#### Optional Settings
SERVICE_NAME // sets the name otherwise it is blank

