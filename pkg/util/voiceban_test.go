package util

import (
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestReadVoiceBans(t *testing.T) {
	f, errOpen := os.Open("../../voice_ban.dt")
	require.NoError(t, errOpen)
	defer f.Close()
	bans, err := ReadVoiceBans(f)
	require.NoError(t, err)
	require.True(t, len(bans) > 0)
}
