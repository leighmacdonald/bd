package ui

import (
	"fyne.io/fyne/v2"
	"github.com/leighmacdonald/steamid/v2/steamid"
)

func SetAvatar(sid64 steamid.SID64, data []byte) {
	if !sid64.Valid() || data == nil {
		return
	}
	avatarCache.Store(sid64.String(), fyne.NewStaticResource(sid64.String(), data))
}

func GetAvatar(sid64 steamid.SID64) fyne.Resource {
	av, found := avatarCache.Load(sid64.String())
	if found {
		return av
	}
	return resourceDefaultavatarJpg
}
