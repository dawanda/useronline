package main

import (
	"fmt"
	"log"
	"sync"
	"time"
)

type TrackKind int

func (k TrackKind) String() string {
	switch k {
	case KindNewUsers:
		return "new"
	case KindRecurring:
		return "recurring"
	default:
		return fmt.Sprintf("%v", int(k))
	}
}

const (
	KindNewUsers TrackKind = iota
	KindRecurring
)

type Tracker struct {
	Debug         bool
	buckets       []map[int]map[string]bool // [kind][bucketID][sessionID]
	bucketMutex   sync.Mutex
	currentBucket int
	statsdClient  *UdpClient
	statsdPrefix  string
	statsdTicker  *time.Ticker
}

func NewTracker(statsdAddr string, statsdPrefix string, debug bool) (*Tracker, error) {
	tracker := &Tracker{Debug: debug}

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
				tracker.getBucketID()
				tracker.bucketMutex.Unlock()
			}
		}()
	}

	tracker.buckets = make([]map[int]map[string]bool, 2)

	for kind := 0; kind < 2; kind++ {
		tracker.buckets[kind] = make(map[int]map[string]bool)
	}

	for minute := 0; minute < 60; minute++ {
		tracker.resetBucket(minute)
	}

	return tracker, nil
}

func (tracker *Tracker) resetBucket(bucketID int) {
	for kind := 0; kind < 2; kind++ {
		for minute := 0; minute < 60; minute++ {
			tracker.buckets[kind][minute] = make(map[string]bool)
		}
	}
}

func (tracker *Tracker) flushReport(bucketID int) {
	var kindStr = []string{"new", "recurring"}
	for kind := 0; kind < 2; kind++ {
		count := len(tracker.buckets[kind][bucketID])
		log.Printf("Tracking summary for %v sessions: %v\n", TrackKind(kind), count)
		if tracker.statsdClient != nil {
			tracker.statsdClient.Sendf("%v.%v:%v|g", tracker.statsdPrefix, kindStr[kind], count)
		}
	}
}

func (tracker *Tracker) getBucketID() int {
	bucketID := time.Now().Minute()
	if bucketID != tracker.currentBucket {
		tracker.flushReport(tracker.currentBucket)
		tracker.resetBucket(bucketID)
		tracker.currentBucket = bucketID
	}
	return bucketID
}

func (tracker *Tracker) Touch(kind TrackKind, sessionID string) {
	tracker.bucketMutex.Lock()
	bid := tracker.getBucketID()
	tracker.buckets[kind][bid][sessionID] = true
	if tracker.Debug {
		log.Printf("Track %v bucket:%v sid:%v count:%v\n", kind, bid, sessionID, len(tracker.buckets[kind][bid]))
	}
	tracker.bucketMutex.Unlock()
}

func (tracker *Tracker) GetCount(kind TrackKind) int {
	tracker.bucketMutex.Lock()
	bid := tracker.getBucketID()
	res := len(tracker.buckets[kind][bid])
	if tracker.Debug {
		log.Printf("GetTrack %v bucket:%v count:%v\n", kind, bid, res)
	}
	tracker.bucketMutex.Unlock()
	return res
}
