package main

import (
	"fmt"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type Tracker struct {
	Name         string
	SessionTTL   time.Duration
	Debug        bool
	sessions     map[string]*time.Timer
	sessionMutex sync.Mutex
	sessionCount int64
}

func NewTracker(name string, sessionTTL time.Duration, statsdAddr *net.UDPAddr, statsdPrefix string, debug bool) *Tracker {
	tracker := &Tracker{Name: name,
		SessionTTL: sessionTTL,
		Debug:      debug,
		sessions:   make(map[string]*time.Timer),
	}

	if statsdAddr != nil {
		go tracker.RunStatsdAgent(statsdAddr, statsdPrefix)
	}

	return tracker
}

func (tracker *Tracker) RunStatsdAgent(addr *net.UDPAddr, statsdPrefix string) {
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		log.Fatalf("Could not create UDP client. %v\n", err)
	}
	defer conn.Close()

	ticker := time.NewTicker(time.Minute * 1)
	for range ticker.C {
		count := tracker.GetCount()
		log.Printf("Tracking summary for %v sessions: %v\n", tracker.Name, count)
		msg := fmt.Sprintf("%v.%v:%v|c", statsdPrefix, tracker.Name, count)
		conn.Write([]byte(msg))
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
