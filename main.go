package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	flag "github.com/ogier/pflag"
)

type Service struct {
	TrackingTTL       time.Duration
	CookieTTL         time.Duration
	SessionTTL        time.Duration
	CookieKey         string
	NewSessions       *Tracker
	RecurringSessions *Tracker
}

func (service *Service) httpTrack(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(service.CookieKey)

	if err != nil {
		sid, err := createSessionID()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Failed to generate Session ID. %v\n", err)
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:    service.CookieKey,
			Value:   sid,
			Expires: time.Now().Add(service.CookieTTL),
		})
		service.NewSessions.Touch(sid)
	} else {
		cookie.Expires = time.Now().Add(service.CookieTTL)
		http.SetCookie(w, cookie)

		if !service.NewSessions.Contains(cookie.Value) {
			service.RecurringSessions.Touch(cookie.Value)
		}
	}

	service.writeEmptyGif(w, r)
}

func (service *Service) httpSessionsCount(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, service.RecurringSessions.GetCount())
}

func (service *Service) httpNewSessionsCount(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, service.NewSessions.GetCount())
}

func (service *Service) httpPing(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "useronline")
}

func (service *Service) httpSessionsToday(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented) // TODO
}

func (service *Service) writeEmptyGif(w http.ResponseWriter, r *http.Request) {
	const base64GifPixel = "R0lGODlhAQABAIAAAP///wAAACwAAAAAAQABAAACAkQBADs="
	w.Header().Set("expires", time.Now().Add(service.TrackingTTL).Format(http.TimeFormat))
	w.Header().Set("cache-control", fmt.Sprintf("max-age=%v", int(service.TrackingTTL.Seconds())))
	w.Header().Set("content-type", "image/gif")
	output, _ := base64.StdEncoding.DecodeString(base64GifPixel)
	io.WriteString(w, string(output))
}

func createSessionID() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}

	uuid := fmt.Sprintf("%X-%X-%X-%X-%X", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
	return uuid, nil
}

func main() {
	var service = Service{
		TrackingTTL: time.Second * 10,
		CookieTTL:   time.Hour * 24 * 30,
		SessionTTL:  time.Minute * 5,
		CookieKey:   "dawanda_uo",
	}

	flag.DurationVar(&service.TrackingTTL, "tracking-ttl", service.TrackingTTL, "tracking pixel expiry timespan")
	flag.DurationVar(&service.CookieTTL, "cookie-ttl", service.CookieTTL, "cookie expiry")
	flag.DurationVar(&service.SessionTTL, "session-ttl", service.SessionTTL, "how long to treat a session as active")
	flag.StringVar(&service.CookieKey, "cookie", service.CookieKey, "Name of the cookie, such as browny")

	httpBindAddr := flag.String("http-bind", "0.0.0.0", "HTTP service bind address")
	httpPort := flag.Int("http-port", 8087, "HTTP service port")
	statsdAddr := flag.String("statsd-addr", "127.0.0.1:8125", "Statsd endpoint")
	statsdPrefix := flag.String("statsd-prefix", "tracker.sessions", "Statsd key prefix")
	debug := flag.Bool("debug", false, "Enable debugging messages")
	flag.Parse()

	tracker, err := NewTracker("recurring", service.SessionTTL, *statsdAddr, *statsdPrefix, *debug)
	if err != nil {
		log.Fatalf("Failed to create tracker. %v\n", err)
	}
	service.RecurringSessions = tracker

	tracker, err = NewTracker("new", service.SessionTTL, *statsdAddr, *statsdPrefix, *debug)
	if err != nil {
		log.Fatalf("Failed to create tracker. %v\n", err)
	}
	service.NewSessions = tracker

	http.HandleFunc("/uo/trck.gif", service.httpTrack)
	http.HandleFunc("/ping", service.httpPing)
	http.HandleFunc("/uo/newsessions/count", service.httpNewSessionsCount)
	http.HandleFunc("/uo/sessions/count", service.httpSessionsCount)
	http.HandleFunc("/uo/sessions/today", service.httpSessionsToday)

	err = http.ListenAndServe(fmt.Sprintf("%v:%v", *httpBindAddr, *httpPort), nil)
	if err != nil {
		log.Fatalf("Failed to start service listener. %v\n", err)
	}
}
