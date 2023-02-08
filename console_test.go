package main

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestParseTimestamp(t *testing.T) {
	t0, err := parseTimestamp("02/05/2023 - 22:51:03")
	require.NoError(t, err)
	require.Equal(t, "2023-02-05 22:51:03 +0000 UTC", t0.String())
}
