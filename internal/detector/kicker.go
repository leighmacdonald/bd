package detector

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const voteCooldown = time.Second * 120

var errAlreadyQueued = errors.New("User already queue for kick")

type KickRequest struct {
	steamID steamid.SID64
	reason  KickReason
	userID  int
}

type kickEntry struct {
	KickRequest
	createdOn time.Time
	attempt   int
}

type Kicker struct {
	queue    []*kickEntry
	mu       sync.RWMutex
	log      *zap.Logger
	kickChan chan KickRequest
}

func NewKicker(logger *zap.Logger) (*Kicker, chan KickRequest) {
	kickChan := make(chan KickRequest)

	return &Kicker{
		log:      logger.Named("kick_queue"),
		kickChan: kickChan,
	}, kickChan
}

func (k *Kicker) Add(sid64 steamid.SID64, userID int, reason KickReason) error {
	k.mu.RLock()

	for _, queuedItem := range k.queue {
		if queuedItem.steamID == sid64 {
			k.mu.RUnlock()

			return errAlreadyQueued
		}
	}

	k.mu.RUnlock()

	k.mu.Lock()

	k.queue = append(k.queue, &kickEntry{
		KickRequest: KickRequest{
			steamID: sid64,
			reason:  reason,
			userID:  userID,
		},
		createdOn: time.Now(),
		attempt:   0,
	})

	k.mu.Unlock()

	return nil
}

func (k *Kicker) empty() bool {
	k.mu.RLock()
	defer k.mu.RUnlock()

	return len(k.queue) == 0
}

func (k *Kicker) Remove(sid64 steamid.SID64) bool {
	var ( //nolint:prealloc
		found    = false
		newQueue []*kickEntry
	)

	k.mu.Lock()

	for _, item := range k.queue {
		if item.steamID == sid64 {
			found = true

			continue
		}

		newQueue = append(newQueue, item)
	}

	k.queue = newQueue

	k.mu.Unlock()

	return found
}

func (k *Kicker) next() *kickEntry {
	if k.empty() {
		return nil
	}

	k.mu.RLock()
	defer k.mu.RUnlock()

	items := k.queue

	sort.SliceStable(items, func(i, j int) bool {
		return items[i].attempt > items[j].attempt
	})

	return items[0]
}

func (k *Kicker) Start(ctx context.Context) {
	lastAttempt := time.Now().Add(-voteCooldown)
	checkTicker := time.NewTicker(time.Second)

	for {
		select {
		case <-checkTicker.C:
			if k.empty() {
				continue
			}

			if time.Since(lastAttempt) < voteCooldown {
				continue
			}

			if nextKick := k.next(); nextKick != nil {
				k.mu.Lock()
				req := nextKick.KickRequest
				nextKick.attempt++
				k.mu.Unlock()

				k.log.Debug("Triggering kick request", zap.String("steam_id", req.steamID.String()))

				k.kickChan <- req
			}

		case <-ctx.Done():
			return
		}
	}
}
