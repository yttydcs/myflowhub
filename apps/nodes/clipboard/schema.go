package clipboard

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/yttydcs/myflowhub/protocol"
)

const (
	SchemaTextEventV1    = "mfh.clipboard.text-event.v1"
	SchemaSendV1         = "mfh.clipboard.send.v1"
	SchemaApplyV1        = "mfh.clipboard.apply.v1"
	SchemaDecisionV1     = "mfh.clipboard.decision.v1"
	SchemaClearV1        = "mfh.clipboard.clear.v1"
	SchemaClearResultV1  = "mfh.clipboard.clear-result.v1"
	SchemaConfigV1       = "mfh.clipboard.config.v1"
	SchemaConfigUpdateV1 = "mfh.clipboard.config-update.v1"
	SchemaStatusV1       = "mfh.clipboard.status.v1"

	ResourceEvents       = "clipboard/events"
	ResourceStatus       = "clipboard/status"
	ResourceConfig       = "clipboard/config"
	ResourceConfigUpdate = "clipboard/config/update"
	CommandSend          = "clipboard/send"
	CommandApply         = "clipboard/apply"
	CommandHistoryClear  = "clipboard/history/clear"

	MaxTextBytes       = 256 << 10
	MaxPendingEvents   = 64
	MaxHistoryEntries  = 4096
	MaxHistoryBytes    = 64 << 20
	MaxHistoryTTLMS    = int64(365 * 24 * 60 * 60 * 1000)
	DefaultHistoryTTL  = int64(30 * 24 * 60 * 60 * 1000)
	DefaultHistorySize = 4 << 20
)

type PeerV1 struct {
	NodeID  string `json:"node_id"`
	Receive bool   `json:"receive"`
}

func (p PeerV1) Validate() error {
	_, err := parseNodeID(p.NodeID)
	return err
}

type ConfigV1 struct {
	Version          int      `json:"version"`
	Revision         uint64   `json:"revision"`
	Enabled          bool     `json:"enabled"`
	MaxInlineBytes   int      `json:"max_inline_bytes"`
	AutoWatch        bool     `json:"auto_watch"`
	AutoApply        bool     `json:"auto_apply"`
	HistoryRetention string   `json:"history_retention"`
	HistoryLimit     int      `json:"history_limit"`
	HistoryMaxBytes  int      `json:"history_max_bytes"`
	HistoryTTLMS     int64    `json:"history_ttl_ms"`
	Peers            []PeerV1 `json:"peers"`
}

func (c ConfigV1) Validate() error {
	if c.Version != 1 || c.Revision == 0 {
		return errors.New("clipboard config version or revision is invalid")
	}
	if c.MaxInlineBytes < 1 || c.MaxInlineBytes > MaxTextBytes {
		return fmt.Errorf("clipboard max_inline_bytes must be between 1 and %d", MaxTextBytes)
	}
	if c.HistoryRetention != "body" && c.HistoryRetention != "metadata" && c.HistoryRetention != "none" {
		return errors.New("clipboard history_retention is invalid")
	}
	if c.HistoryLimit < 1 || c.HistoryLimit > MaxHistoryEntries {
		return fmt.Errorf("clipboard history_limit must be between 1 and %d", MaxHistoryEntries)
	}
	if c.HistoryMaxBytes < 1 || c.HistoryMaxBytes > MaxHistoryBytes {
		return fmt.Errorf("clipboard history_max_bytes must be between 1 and %d", MaxHistoryBytes)
	}
	if c.HistoryTTLMS < 1 || c.HistoryTTLMS > MaxHistoryTTLMS {
		return errors.New("clipboard history_ttl_ms is invalid")
	}
	if len(c.Peers) > protocol.MaxItems {
		return errors.New("clipboard peers exceed protocol item limit")
	}
	var previous protocol.NodeID
	for index, peer := range c.Peers {
		current, err := parseNodeID(peer.NodeID)
		if err != nil {
			return fmt.Errorf("peers[%d]: %w", index, err)
		}
		if index > 0 && previous >= current {
			return errors.New("clipboard peers must be unique and sorted by canonical NodeID")
		}
		previous = current
	}
	return nil
}

type ConfigUpdateV1 struct {
	Version          int      `json:"version"`
	ExpectedRevision uint64   `json:"expected_revision"`
	Enabled          bool     `json:"enabled"`
	MaxInlineBytes   int      `json:"max_inline_bytes"`
	AutoWatch        bool     `json:"auto_watch"`
	AutoApply        bool     `json:"auto_apply"`
	HistoryRetention string   `json:"history_retention"`
	HistoryLimit     int      `json:"history_limit"`
	HistoryMaxBytes  int      `json:"history_max_bytes"`
	HistoryTTLMS     int64    `json:"history_ttl_ms"`
	Peers            []PeerV1 `json:"peers"`
}

func (u ConfigUpdateV1) Validate() error {
	if u.Version != 1 || u.ExpectedRevision == 0 {
		return errors.New("clipboard config update version or revision is invalid")
	}
	peers := append([]PeerV1(nil), u.Peers...)
	sortPeers(peers)
	return ConfigV1{
		Version: 1, Revision: u.ExpectedRevision, Enabled: u.Enabled, MaxInlineBytes: u.MaxInlineBytes,
		AutoWatch: u.AutoWatch, AutoApply: u.AutoApply, HistoryRetention: u.HistoryRetention,
		HistoryLimit: u.HistoryLimit, HistoryMaxBytes: u.HistoryMaxBytes, HistoryTTLMS: u.HistoryTTLMS, Peers: peers,
	}.Validate()
}

type TextEventV1 struct {
	Version         int    `json:"version"`
	EventID         string `json:"event_id"`
	OriginNodeID    string `json:"origin_node_id"`
	CreatedAtUnixMS int64  `json:"created_at_unix_ms"`
	ContentType     string `json:"content_type"`
	Text            string `json:"text"`
	SizeBytes       int    `json:"size_bytes"`
	SHA256          string `json:"sha256"`
}

func (e TextEventV1) Validate() error {
	if e.Version != 1 || !validHex(e.EventID, 16) {
		return errors.New("clipboard event version or event_id is invalid")
	}
	if _, err := parseNodeID(e.OriginNodeID); err != nil {
		return fmt.Errorf("clipboard origin_node_id: %w", err)
	}
	if e.CreatedAtUnixMS <= 0 || e.ContentType != "text/plain; charset=utf-8" {
		return errors.New("clipboard event timestamp or content type is invalid")
	}
	if !utf8.ValidString(e.Text) || len(e.Text) == 0 || len(e.Text) > MaxTextBytes || e.SizeBytes != len([]byte(e.Text)) {
		return errors.New("clipboard event text or byte size is invalid")
	}
	if e.SHA256 != textHash(e.Text) {
		return errors.New("clipboard event SHA-256 does not match text")
	}
	return nil
}

type SendV1 struct {
	Version int    `json:"version"`
	Text    string `json:"text"`
}

func (s SendV1) Validate() error {
	if s.Version != 1 || !utf8.ValidString(s.Text) || len(s.Text) == 0 || len(s.Text) > MaxTextBytes {
		return errors.New("clipboard send payload is invalid")
	}
	return nil
}

type ApplyV1 struct {
	Version int    `json:"version"`
	EventID string `json:"event_id"`
}

func (a ApplyV1) Validate() error {
	if a.Version != 1 || !validHex(a.EventID, 16) {
		return errors.New("clipboard apply payload is invalid")
	}
	return nil
}

type ClearV1 struct {
	Version int `json:"version"`
}

func (c ClearV1) Validate() error {
	if c.Version != 1 {
		return errors.New("clipboard clear version must be 1")
	}
	return nil
}

type ClearResultV1 struct {
	Version int `json:"version"`
	Removed int `json:"removed"`
}

func (r ClearResultV1) Validate() error {
	if r.Version != 1 || r.Removed < 0 || r.Removed > MaxHistoryEntries {
		return errors.New("clipboard clear result is invalid")
	}
	return nil
}

type DecisionV1 struct {
	Version     int    `json:"version"`
	Action      string `json:"action"`
	EventID     string `json:"event_id,omitempty"`
	OriginNode  string `json:"origin_node_id,omitempty"`
	SizeBytes   int    `json:"size_bytes,omitempty"`
	HashPrefix  string `json:"hash_prefix,omitempty"`
	Reason      string `json:"reason,omitempty"`
	TimestampMS int64  `json:"timestamp_ms"`
}

func (d DecisionV1) Validate() error {
	allowed := d.Action == "published" || d.Action == "pending" || d.Action == "applied" || d.Action == "ignored"
	if d.Version != 1 || !allowed || d.TimestampMS <= 0 || d.SizeBytes < 0 || len(d.Reason) > 1024 {
		return errors.New("clipboard decision is invalid")
	}
	if d.EventID != "" && !validHex(d.EventID, 16) {
		return errors.New("clipboard decision event_id is invalid")
	}
	if d.OriginNode != "" {
		if _, err := parseNodeID(d.OriginNode); err != nil {
			return err
		}
	}
	if d.HashPrefix != "" && (len(d.HashPrefix) != 12 || !validHex(d.HashPrefix, 6)) {
		return errors.New("clipboard decision hash prefix is invalid")
	}
	return nil
}

type StatusV1 struct {
	Version          int    `json:"version"`
	ConfigRevision   uint64 `json:"config_revision"`
	Enabled          bool   `json:"enabled"`
	AutoWatch        bool   `json:"auto_watch"`
	AutoApply        bool   `json:"auto_apply"`
	PeerCount        int    `json:"peer_count"`
	PendingCount     int    `json:"pending_count"`
	PendingEventID   string `json:"pending_event_id,omitempty"`
	PendingSizeBytes int    `json:"pending_size_bytes,omitempty"`
	PendingHash      string `json:"pending_hash_prefix,omitempty"`
	HistoryCount     int    `json:"history_count"`
	HistoryBodyBytes int    `json:"history_body_bytes"`
	LastAction       string `json:"last_action,omitempty"`
	LastEventID      string `json:"last_event_id,omitempty"`
	LastSizeBytes    int    `json:"last_size_bytes,omitempty"`
	LastHashPrefix   string `json:"last_hash_prefix,omitempty"`
	LastError        string `json:"last_error,omitempty"`
	ObservedAtUnixMS int64  `json:"observed_at_unix_ms"`
}

func (s StatusV1) Validate() error {
	if s.Version != 1 || s.ConfigRevision == 0 || s.PeerCount < 0 || s.PendingCount < 0 || s.PendingCount > MaxPendingEvents ||
		s.PendingSizeBytes < 0 || s.HistoryCount < 0 || s.HistoryCount > MaxHistoryEntries || s.HistoryBodyBytes < 0 ||
		s.LastSizeBytes < 0 || s.ObservedAtUnixMS <= 0 || len(s.LastError) > 1024 {
		return errors.New("clipboard status is invalid")
	}
	if strings.Contains(s.LastError, "\n") || strings.Contains(s.LastError, "\r") {
		return errors.New("clipboard status error must be one line")
	}
	return nil
}

func DefaultConfig() ConfigV1 {
	return ConfigV1{
		Version: 1, Revision: 1, Enabled: false, MaxInlineBytes: 64 << 10,
		AutoWatch: false, AutoApply: false, HistoryRetention: "body", HistoryLimit: 256,
		HistoryMaxBytes: DefaultHistorySize, HistoryTTLMS: DefaultHistoryTTL, Peers: []PeerV1{},
	}
}

func configFromUpdate(revision uint64, update ConfigUpdateV1) ConfigV1 {
	peers := append([]PeerV1(nil), update.Peers...)
	sortPeers(peers)
	return ConfigV1{
		Version: 1, Revision: revision, Enabled: update.Enabled, MaxInlineBytes: update.MaxInlineBytes,
		AutoWatch: update.AutoWatch, AutoApply: update.AutoApply, HistoryRetention: update.HistoryRetention,
		HistoryLimit: update.HistoryLimit, HistoryMaxBytes: update.HistoryMaxBytes, HistoryTTLMS: update.HistoryTTLMS, Peers: peers,
	}
}

func cloneConfig(value ConfigV1) ConfigV1 {
	value.Peers = append([]PeerV1(nil), value.Peers...)
	return value
}

func sortPeers(peers []PeerV1) {
	sort.Slice(peers, func(i, j int) bool {
		left, _ := strconv.ParseUint(peers[i].NodeID, 10, 64)
		right, _ := strconv.ParseUint(peers[j].NodeID, 10, 64)
		return left < right
	})
}

func parseNodeID(value string) (protocol.NodeID, error) {
	if value == "" || strings.HasPrefix(value, "+") || (len(value) > 1 && strings.HasPrefix(value, "0")) {
		return 0, errors.New("NodeID must be a canonical non-zero decimal value")
	}
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil || parsed == 0 {
		return 0, errors.New("NodeID must be a canonical non-zero decimal value")
	}
	return protocol.NodeID(parsed), nil
}

func formatNodeID(value protocol.NodeID) string { return strconv.FormatUint(uint64(value), 10) }

func textHash(text string) string {
	digest := sha256.Sum256([]byte(text))
	return hex.EncodeToString(digest[:])
}

func hashPrefix(hash string) string {
	if len(hash) < 12 {
		return ""
	}
	return hash[:12]
}

func validHex(value string, bytes int) bool {
	if len(value) != bytes*2 || value != strings.ToLower(value) {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == bytes
}
