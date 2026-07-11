package engine

import (
	"encoding/json"
	"fmt"
	"log"

	bolt "go.etcd.io/bbolt"
)

// Store handles async persistence to bbolt. Never blocks the hot path.
type Store struct {
	db      *bolt.DB
	alertCh chan Alert
	trackCh chan AISMessage
	done    chan struct{}
}

var (
	alertBucket = []byte("alerts")
	trackBucket = []byte("tracks")
)

// NewStore creates a new bbolt store with async write goroutines.
func NewStore(path string, bufferSize int) (*Store, error) {
	db, err := bolt.Open(path, 0600, &bolt.Options{NoSync: true}) // ponytail: NoSync for throughput, data loss on crash is acceptable for demo
	if err != nil {
		return nil, fmt.Errorf("open bbolt: %w", err)
	}

	// Create buckets
	err = db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(alertBucket); err != nil {
			return err
		}
		_, err := tx.CreateBucketIfNotExists(trackBucket)
		return err
	})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("create buckets: %w", err)
	}

	s := &Store{
		db:      db,
		alertCh: make(chan Alert, bufferSize),
		trackCh: make(chan AISMessage, bufferSize),
		done:    make(chan struct{}),
	}

	go s.writeAlerts()
	go s.writeTracks()

	return s, nil
}

// WriteAlert queues an alert for async persistence.
func (s *Store) WriteAlert(a Alert) {
	select {
	case s.alertCh <- a:
	default:
		// ponytail: drop if buffer full, never block hot path
	}
}

// WriteTrack queues a track point for async persistence.
func (s *Store) WriteTrack(msg AISMessage) {
	select {
	case s.trackCh <- msg:
	default:
		// ponytail: drop if buffer full
	}
}

func (s *Store) writeAlerts() {
	for a := range s.alertCh {
		data, err := json.Marshal(a)
		if err != nil {
			log.Printf("store: marshal alert: %v", err)
			continue
		}
		err = s.db.Update(func(tx *bolt.Tx) error {
			b := tx.Bucket(alertBucket)
			return b.Put([]byte(a.AlertID), data)
		})
		if err != nil {
			log.Printf("store: write alert: %v", err)
		}
	}
}

func (s *Store) writeTracks() {
	for msg := range s.trackCh {
		data, err := json.Marshal(msg)
		if err != nil {
			continue
		}
		key := fmt.Sprintf("%s-%d", msg.VesselID, msg.TimestampMs)
		err = s.db.Update(func(tx *bolt.Tx) error {
			b := tx.Bucket(trackBucket)
			return b.Put([]byte(key), data)
		})
		if err != nil {
			log.Printf("store: write track: %v", err)
		}
	}
}

// Close stops write goroutines and closes the database.
func (s *Store) Close() {
	close(s.alertCh)
	close(s.trackCh)
	s.db.Close()
}
