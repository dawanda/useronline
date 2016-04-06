## UserOnline Tracker Service

User-Online Tracker Service, written in Go programming language.

### Usage

```
docker build -t uo .
docker run --rm --net=host uo -http-port 8080 -stats-addr 127.0.0.1:8125
```
