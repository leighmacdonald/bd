package ui

import (
	"fyne.io/fyne/v2"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"sync"
)

type avatarCache struct {
	*sync.RWMutex
	userAvatar map[steamid.SID64]fyne.Resource
}

func (cache *avatarCache) SetAvatar(sid64 steamid.SID64, data []byte) {
	if !sid64.Valid() || data == nil {
		return
	}
	cache.Lock()
	cache.userAvatar[sid64] = fyne.NewStaticResource(sid64.String(), data)
	cache.Unlock()
}

func (cache *avatarCache) GetAvatar(sid64 steamid.SID64) fyne.Resource {
	cache.RLock()
	defer cache.RUnlock()
	av, found := cache.userAvatar[sid64]
	if found {
		return av
	}
	return resourceDefaultavatarJpg
}
