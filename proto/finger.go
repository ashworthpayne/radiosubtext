package proto

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type FingerEntry struct {
	Callsign     string    `json:"callsign"`
	LastResponse string    `json:"last_response"`
	Updated      time.Time `json:"updated"`
}

type FingerCache map[string]FingerEntry

func getCachePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".radiosubtext", "finger.json")
}

func LoadFingerCache() (FingerCache, error) {
	path := getCachePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return FingerCache{}, nil // empty cache
		}
		return nil, err
	}
	var cache FingerCache
	err = json.Unmarshal(data, &cache)
	return cache, err
}

func SaveFingerCache(cache FingerCache) error {
	path := getCachePath()
	os.MkdirAll(filepath.Dir(path), 0755)
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
