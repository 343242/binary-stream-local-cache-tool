package writer

import (
	"math"
	"strings"
	"sync"
	"time"
)

type Event struct {
	Kind            string `json:"kind"`
	Message         string `json:"message"`
	TimestampUnixMs int64  `json:"timestampUnixMs"`
}

// EventFeed keeps a recent coarse-grained writer event history for the GUI.
type EventFeed struct {
	mu           sync.Mutex
	events       []Event
	maxSize      int
	limiter      *tokenBucket
	dedupeWindow time.Duration
	lastSeen     map[string]time.Time
	now          func() time.Time
}

func NewEventFeed(maxSize int, ratePerSecond int, dedupeWindow time.Duration) *EventFeed {
	if maxSize <= 0 {
		maxSize = 1
	}
	return &EventFeed{
		events:       make([]Event, 0, maxSize),
		maxSize:      maxSize,
		limiter:      newTokenBucket(ratePerSecond),
		dedupeWindow: dedupeWindow,
		lastSeen:     make(map[string]time.Time),
		now:          time.Now,
	}
}

func (f *EventFeed) Emit(event Event) {
	f.mu.Lock()
	defer f.mu.Unlock()

	now := f.now()
	if event.TimestampUnixMs == 0 {
		event.TimestampUnixMs = now.UnixMilli()
	}
	if !f.limiter.Allow(now) {
		return
	}
	if f.shouldDropDuplicate(event, now) {
		return
	}

	if len(f.events) == f.maxSize {
		copy(f.events, f.events[1:])
		f.events[len(f.events)-1] = event
		return
	}
	f.events = append(f.events, event)
}

func (f *EventFeed) Snapshot() []Event {
	f.mu.Lock()
	defer f.mu.Unlock()

	events := make([]Event, len(f.events))
	copy(events, f.events)
	return events
}

func (f *EventFeed) shouldDropDuplicate(event Event, now time.Time) bool {
	if f.dedupeWindow <= 0 || !isWarningKind(event.Kind) {
		return false
	}

	key := event.Kind + "\x00" + event.Message
	if last, ok := f.lastSeen[key]; ok && now.Sub(last) < f.dedupeWindow {
		return true
	}
	f.lastSeen[key] = now
	return false
}

func isWarningKind(kind string) bool {
	kind = strings.ToLower(kind)
	return kind == "warning" || strings.Contains(kind, "warning")
}

type tokenBucket struct {
	rate   float64
	burst  float64
	tokens float64
	last   time.Time
}

func newTokenBucket(ratePerSecond int) *tokenBucket {
	if ratePerSecond <= 0 {
		return &tokenBucket{
			rate:   0,
			burst:  math.Inf(1),
			tokens: math.Inf(1),
			last:   time.Now(),
		}
	}
	burst := float64(ratePerSecond)
	return &tokenBucket{
		rate:   float64(ratePerSecond),
		burst:  burst,
		tokens: burst,
		last:   time.Now(),
	}
}

func (b *tokenBucket) Allow(now time.Time) bool {
	if b.rate <= 0 {
		return true
	}

	elapsed := now.Sub(b.last).Seconds()
	if elapsed > 0 {
		b.tokens = math.Min(b.burst, b.tokens+(elapsed*b.rate))
		b.last = now
	}
	if b.tokens < 1 {
		return false
	}

	b.tokens--
	return true
}
