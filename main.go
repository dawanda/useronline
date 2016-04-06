package main

import (
	"crypto/rand"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const (
	cookieKey   string        = "dawanda_uo"
	trackingTTL time.Duration = time.Second * 10
	cookieTTL   time.Duration = time.Hour * 24 * 30
)

func httpTrack(w http.ResponseWriter, r *http.Request, tracker *Tracker) {
	cookie, err := r.Cookie(cookieKey)

	if err != nil {
		sid, err := createSessionID()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Failed to generate Session ID. %v\n", err)
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:    cookieKey,
			Value:   sid,
			Expires: time.Now().Add(cookieTTL),
		})
		tracker.Touch(KindNewUsers, sid)
	} else {
		tracker.Touch(KindRecurring, cookie.Value)
	}

	writeEmptyGif(w, r)
}

func httpNotImplemented(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
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

func writeEmptyGif(w http.ResponseWriter, r *http.Request) {
	const base64GifPixel = "R0lGODlhAQABAIAAAP///wAAACwAAAAAAQABAAACAkQBADs="
	w.Header().Set("expires", time.Now().Add(trackingTTL).Format(http.TimeFormat))
	w.Header().Set("cache-control", fmt.Sprintf("max-age=%v", int(trackingTTL.Seconds())))
	w.Header().Set("content-type", "image/gif")
	output, _ := base64.StdEncoding.DecodeString(base64GifPixel)
	io.WriteString(w, string(output))
}

func main() {
	httpBindAddr := flag.String("http-bind", "0.0.0.0", "HTTP service bind address")
	httpPort := flag.Int("http-port", 8087, "HTTP service port")
	statsdAddr := flag.String("statsd-addr", "127.0.0.1:8125", "Statsd endpoint")
	statsdPrefix := flag.String("statsd-prefix", "tracker.sessions", "Statsd key prefix")
	debug := flag.Bool("debug", false, "Enable debugging messages")
	flag.Parse()

	tracker, err := NewTracker(*statsdAddr, *statsdPrefix, *debug)
	if err != nil {
		log.Fatalf("Failed to create tracker. %v\n", err)
	}

	http.HandleFunc("/uo/trck.gif", func(w http.ResponseWriter, r *http.Request) {
		httpTrack(w, r, tracker)
	})

	http.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "useronline\n")
	})

	http.HandleFunc("/uo/newsessions/count", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, tracker.GetCount(KindNewUsers))
	})

	http.HandleFunc("/uo/sessions/count", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, tracker.GetCount(KindRecurring))
	})

	http.HandleFunc("/uo/sessions/today", httpNotImplemented)

	err = http.ListenAndServe(fmt.Sprintf("%v:%v", *httpBindAddr, *httpPort), nil)
	if err != nil {
		log.Fatalf("Failed to start service listener. %v\n", err)
	}
}
