package fileserver

import (
	"encoding/json"
	"net/http"
	"time"
)

type EtagStore struct {
	store    map[string]string
	interval time.Duration
}

func NewEtagStore() *EtagStore {
	s := &EtagStore{
		store:    make(map[string]string),
		interval: 10 * time.Minute,
	}
	go s.sync()

	return s
}

func (s *EtagStore) fetch(key string) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(http.MethodHead, key, nil)
	if err != nil {
		return
	}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK && resp.Header.Get("Etag") != "" {
		s.store[key] = resp.Header.Get("Etag")
	}
}

func (s *EtagStore) sync() {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for range ticker.C {
		for key := range s.store {
			go s.fetch(key)
		}
	}
}

func (s *EtagStore) Set(key string, etag string) {
	s.store[key] = etag
}

func (s *EtagStore) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.store)
}

func (s *EtagStore) UnmarshalJSON(data []byte) error {
	var store map[string]string
	if err := json.Unmarshal(data, &store); err != nil {
		return err
	}
	s.store = store
	return nil
}

func (s *EtagStore) MergeJSON(o string) ([]byte, error) {
	var other map[string]string

	if err := json.Unmarshal([]byte(o), &other); err != nil {
		return nil, err
	}
	for k, v := range s.store {
		other[k] = v
	}
	return json.Marshal(other)
}
