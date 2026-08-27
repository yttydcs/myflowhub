package clipboard

import (
	"errors"
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/yttydcs/myflowhub/internal/keystore"
)

type HistoryEntry struct {
	EventID         string `json:"event_id"`
	Kind            string `json:"kind"`
	OriginNodeID    string `json:"origin_node_id"`
	Text            string `json:"text,omitempty"`
	SizeBytes       int    `json:"size_bytes"`
	SHA256          string `json:"sha256"`
	TimestampUnixMS int64  `json:"timestamp_unix_ms"`
}

type historyState struct {
	Version int            `json:"version"`
	Entries []HistoryEntry `json:"entries"`
}

type historyStore struct {
	store   *keystore.Store
	entries []HistoryEntry
}

func loadHistory(store *keystore.Store, config ConfigV1, now time.Time) (*historyStore, error) {
	if store == nil {
		return nil, errors.New("clipboard history requires a durable store")
	}
	state := historyState{Version: 1, Entries: []HistoryEntry{}}
	_, err := store.Load("clipboard_history.json", &state)
	if err != nil {
		return nil, fmt.Errorf("load clipboard history: %w", err)
	}
	if state.Version != 1 {
		return nil, errors.New("clipboard history version is invalid")
	}
	value := &historyStore{store: store, entries: append([]HistoryEntry(nil), state.Entries...)}
	if err := value.normalize(config, now); err != nil {
		return nil, err
	}
	return value, nil
}

func (h *historyStore) add(config ConfigV1, event TextEventV1, kind string, now time.Time) error {
	if config.HistoryRetention == "none" {
		return nil
	}
	entry := HistoryEntry{
		EventID: event.EventID, Kind: kind, OriginNodeID: event.OriginNodeID, SizeBytes: event.SizeBytes,
		SHA256: event.SHA256, TimestampUnixMS: now.UTC().UnixMilli(),
	}
	if config.HistoryRetention == "body" {
		entry.Text = event.Text
	}
	filtered := h.entries[:0]
	for _, current := range h.entries {
		if current.EventID != entry.EventID && current.SHA256 != entry.SHA256 {
			filtered = append(filtered, current)
		}
	}
	h.entries = append([]HistoryEntry{entry}, filtered...)
	return h.normalize(config, now)
}

func (h *historyStore) clear() (int, error) {
	removed := len(h.entries)
	h.entries = nil
	return removed, h.persist()
}

func (h *historyStore) snapshot() []HistoryEntry {
	return append([]HistoryEntry(nil), h.entries...)
}

func (h *historyStore) stats() (int, int) {
	bytes := 0
	for _, entry := range h.entries {
		bytes += len([]byte(entry.Text))
	}
	return len(h.entries), bytes
}

func (h *historyStore) normalize(config ConfigV1, now time.Time) error {
	cutoff := now.UTC().UnixMilli() - config.HistoryTTLMS
	result := make([]HistoryEntry, 0, len(h.entries))
	bodyBytes := 0
	seen := make(map[string]struct{}, len(h.entries))
	for _, entry := range h.entries {
		if len(result) >= config.HistoryLimit || entry.TimestampUnixMS < cutoff || entry.TimestampUnixMS <= 0 ||
			!validHex(entry.EventID, 16) || !validHex(entry.SHA256, 32) || entry.SizeBytes < 1 || entry.SizeBytes > MaxTextBytes {
			continue
		}
		if _, duplicate := seen[entry.SHA256]; duplicate {
			continue
		}
		seen[entry.SHA256] = struct{}{}
		if config.HistoryRetention != "body" {
			entry.Text = ""
		} else {
			if !utf8BodyMatches(entry) || bodyBytes+len([]byte(entry.Text)) > config.HistoryMaxBytes {
				continue
			}
			bodyBytes += len([]byte(entry.Text))
		}
		result = append(result, entry)
	}
	if config.HistoryRetention == "none" {
		result = nil
	}
	h.entries = result
	return h.persist()
}

func (h *historyStore) persist() error {
	if err := h.store.Save("clipboard_history.json", historyState{Version: 1, Entries: h.entries}); err != nil {
		return fmt.Errorf("persist clipboard history: %w", err)
	}
	return nil
}

func utf8BodyMatches(entry HistoryEntry) bool {
	return entry.Text != "" && utf8.ValidString(entry.Text) && len([]byte(entry.Text)) == entry.SizeBytes && textHash(entry.Text) == entry.SHA256
}
