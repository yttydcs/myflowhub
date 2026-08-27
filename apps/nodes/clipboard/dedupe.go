package clipboard

import "time"

type recentSet struct {
	limit int
	ttl   time.Duration
	items map[string]time.Time
	order []string
}

func newRecentSet(limit int, ttl time.Duration) *recentSet {
	return &recentSet{limit: limit, ttl: ttl, items: make(map[string]time.Time)}
}

func (s *recentSet) Contains(value string, now time.Time) bool {
	s.trim(now)
	_, ok := s.items[value]
	return ok
}

func (s *recentSet) Add(value string, now time.Time) {
	if value == "" {
		return
	}
	s.trim(now)
	if _, exists := s.items[value]; !exists {
		s.order = append(s.order, value)
	}
	s.items[value] = now
	s.trim(now)
}

func (s *recentSet) Consume(value string, now time.Time) bool {
	if !s.Contains(value, now) {
		return false
	}
	delete(s.items, value)
	return true
}

func (s *recentSet) Remove(value string) { delete(s.items, value) }

func (s *recentSet) trim(now time.Time) {
	for len(s.order) > 0 {
		oldest := s.order[0]
		observed, exists := s.items[oldest]
		if exists && len(s.items) <= s.limit && now.Sub(observed) <= s.ttl {
			break
		}
		s.order = s.order[1:]
		if exists && (len(s.items) > s.limit || now.Sub(observed) > s.ttl) {
			delete(s.items, oldest)
		}
	}
}
