## UserOnline Tracker Service

User-Online Tracker Service, written in Go programming language.

### Usage

```
docker build -t uo .
docker run --rm --net=host uo -http-port 8080 -stats-addr 127.0.0.1:8125
```

### Endpoint API

* `/ping`: ping health check
* `/uo/trck.gif`: tracking pixel to be included in your page
* `/uo/sessions/count`: reads out the count of recurring sessions
* `/uo/newsessions/count`: reads out the count of new sessions
