package main

import (
	"log"
	"sync"
	"time"
)

type Tracker struct {
	Name          string
	SessionTTL    time.Duration
	Debug         bool
	buckets       map[string]*time.Timer
	bucketMutex   sync.Mutex
	currentBucket int
	statsdClient  *UdpClient
	statsdPrefix  string
	statsdTicker  *time.Ticker
}

func NewTracker(name string, sessionTTL time.Duration, statsdAddr string, statsdPrefix string, debug bool) (*Tracker, error) {
	tracker := &Tracker{Name: name, SessionTTL: sessionTTL, Debug: debug}

	if len(statsdAddr) > 0 {
		udpClient, err := NewUdpClient(statsdAddr)
		if err != nil {
			return nil, err
		}
		tracker.statsdClient = udpClient

		tracker.statsdTicker = time.NewTicker(time.Minute * 1)
		go func() {
			for range tracker.statsdTicker.C {
				// getting the current bucket ID triggers an implicit flush, if needed
				tracker.bucketMutex.Lock()
				tracker.flushReport()
				tracker.bucketMutex.Unlock()
			}
		}()
	}

	tracker.buckets = make(map[string]*time.Timer)

	return tracker, nil
}

func (tracker *Tracker) Run() {
	// TODO?
}

func (tracker *Tracker) flushReport() {
	count := tracker.GetCount()
	log.Printf("Tracking summary for %v sessions: %v\n", tracker.Name, count)
	if tracker.statsdClient != nil {
		tracker.statsdClient.Sendf("%v.%v:%v|c", tracker.statsdPrefix, tracker.Name, count)
	}
}

func (tracker *Tracker) Debugf(msg string, args ...interface{}) {
	if tracker.Debug {
		log.Printf(msg, args...)
	}
}

func (tracker *Tracker) Touch(sessionID string) {
	tracker.bucketMutex.Lock()

	t, ok := tracker.buckets[sessionID]
	if ok {
		t.Reset(tracker.SessionTTL)
	} else {
		t := time.NewTimer(tracker.SessionTTL)
		tracker.buckets[sessionID] = t
		start := time.Now()
		go func() {
			<-t.C
			tracker.Debugf("Session %v timed out after %v\n", sessionID, time.Now().Sub(start))
			tracker.bucketMutex.Lock()
			delete(tracker.buckets, sessionID)
			tracker.bucketMutex.Unlock()
		}()
	}

	tracker.Debugf("Track %v sid:%v count:%v\n", tracker.Name, sessionID, len(tracker.buckets))
	tracker.bucketMutex.Unlock()
}

func (tracker *Tracker) GetCount() int {
	tracker.bucketMutex.Lock()
	res := len(tracker.buckets)
	tracker.Debugf("GetTrack %v count:%v\n", tracker.Name, res)
	tracker.bucketMutex.Unlock()
	return res
}
