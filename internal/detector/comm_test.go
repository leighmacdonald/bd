package detector_test

import (
	"testing"

	"github.com/leighmacdonald/bd/internal/detector"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	s := `CTFLobbyShared: ID:00021f0d433926bb  13 member(s), 1 pending
  Member[0] [U:1:1176385561]  team = TF_GC_TEAM_INVADERS  type = MATCH_PLAYER
  Member[1] [U:1:32604711]  team = TF_GC_TEAM_DEFENDERS  type = MATCH_PLAYER
  Member[2] [U:1:123868297]  team = TF_GC_TEAM_INVADERS  type = MATCH_PLAYER
  Member[3] [U:1:82676318]  team = TF_GC_TEAM_DEFENDERS  type = MATCH_PLAYER
  Member[4] [U:1:1378752]  team = TF_GC_TEAM_DEFENDERS  type = MATCH_PLAYER
  Member[5] [U:1:70950033]  team = TF_GC_TEAM_INVADERS  type = MATCH_PLAYER
  Member[6] [U:1:83786318]  team = TF_GC_TEAM_INVADERS  type = MATCH_PLAYER
  Member[7] [U:1:888864434]  team = TF_GC_TEAM_DEFENDERS  type = MATCH_PLAYER
  Member[8] [U:1:1215529789]  team = TF_GC_TEAM_DEFENDERS  type = MATCH_PLAYER
  Member[9] [U:1:1215347819]  team = TF_GC_TEAM_INVADERS  type = MATCH_PLAYER
  Member[10] [U:1:30412989]  team = TF_GC_TEAM_DEFENDERS  type = MATCH_PLAYER
  Member[11] [U:1:265044852]  team = TF_GC_TEAM_DEFENDERS  type = MATCH_PLAYER
  Member[12] [U:1:1216086694]  team = TF_GC_TEAM_INVADERS  type = MATCH_PLAYER
  Pending[0] [U:1:1201744457]  team = TF_GC_TEAM_INVADERS  type = MATCH_PLAYER

`
	require.Equal(t, 14, len(detector.ParseLobbyPlayers(s)))
}
