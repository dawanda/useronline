package main

import (
	"log"
	"sync"
	"sync/atomic"
	"time"
)

type Tracker struct {
	Name         string
	SessionTTL   time.Duration
	StatsdPrefix string
	Debug        bool
	sessions     map[string]*time.Timer
	sessionMutex sync.Mutex
	sessionCount int64
	statsdClient *UdpClient
	statsdTicker *time.Ticker
}

func NewTracker(name string, sessionTTL time.Duration, statsdAddr string, statsdPrefix string, debug bool) (*Tracker, error) {
	tracker := &Tracker{Name: name,
		SessionTTL:   sessionTTL,
		StatsdPrefix: statsdPrefix,
		Debug:        debug,
	}

	if len(statsdAddr) > 0 {
		udpClient, err := NewUdpClient(statsdAddr)
		if err != nil {
			return nil, err
		}
		tracker.statsdClient = udpClient

		tracker.statsdTicker = time.NewTicker(time.Minute * 1)
		go func() {
			for range tracker.statsdTicker.C {
				tracker.FlushReport()
			}
		}()
	}

	tracker.sessions = make(map[string]*time.Timer)

	return tracker, nil
}

func (tracker *Tracker) FlushReport() {
	count := tracker.GetCount()
	log.Printf("Tracking summary for %v sessions: %v\n", tracker.Name, count)
	if tracker.statsdClient != nil {
		tracker.statsdClient.Sendf("%v.%v:%v|c", tracker.StatsdPrefix, tracker.Name, count)
	}
}

func (tracker *Tracker) Debugf(msg string, args ...interface{}) {
	if tracker.Debug {
		log.Printf(msg, args...)
	}
}

func (tracker *Tracker) Contains(sessionID string) bool {
	tracker.sessionMutex.Lock()
	_, ok := tracker.sessions[sessionID]
	tracker.sessionMutex.Unlock()
	return ok
}

func (tracker *Tracker) Touch(sessionID string) {
	tracker.sessionMutex.Lock()

	t, ok := tracker.sessions[sessionID]
	if ok {
		t.Reset(tracker.SessionTTL)
	} else {
		t := time.NewTimer(tracker.SessionTTL)
		tracker.sessions[sessionID] = t
		start := time.Now()
		atomic.AddInt64(&tracker.sessionCount, 1)
		go func() {
			<-t.C
			atomic.AddInt64(&tracker.sessionCount, -1)
			tracker.Debugf("Session %v timed out after %v\n", sessionID, time.Now().Sub(start))
			tracker.sessionMutex.Lock()
			delete(tracker.sessions, sessionID)
			tracker.sessionMutex.Unlock()
		}()
	}

	tracker.Debugf("Track %v sid:%v count:%v\n", tracker.Name, sessionID, len(tracker.sessions))
	tracker.sessionMutex.Unlock()
}

func (tracker *Tracker) GetCount() int64 {
	res := atomic.LoadInt64(&tracker.sessionCount)
	tracker.Debugf("GetTrack %v count:%v\n", tracker.Name, res)
	return res
}
