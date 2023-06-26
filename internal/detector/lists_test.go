package detector

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestListParse(t *testing.T) {
	badSchema := []byte(`{
    "$schema": "https://raw.githubusercontent.com/PazerOP/tf2_bot_detector/master/schemas/v3/playerlist.schema.json",
    "file_info": {
        "authors": [
            "local"
        ],
        "description": "local player list",
        "title": "local",
        "update_url": ""
    },
    "players": [
        {
            "attributes": [
                "bot"
            ],
            "last_seen": {
                "player_name": "personman",
                "time": 1677390631
            },
            "steamid": "76561199006548700"
        },
        {
            "attributes": [
                "bot"
            ],
            "last_seen": {
                "player_name": "ุ",
                "time": 1678491214
            },
            "steamid": 76561198084134025
        },
        {
            "attributes": [
                "bot"
            ],
            "last_seen": {
                "player_name": "x",
                "time": 1677390631
            },
			"steamid": 76561199006548705}
    ]
}
`)
	goodSchema := []byte(`{
    "$schema": "https://raw.githubusercontent.com/PazerOP/tf2_bot_detector/master/schemas/v3/playerlist.schema.json",
    "file_info": {
        "authors": [
            "local"
        ],
        "description": "local player list",
        "title": "local",
        "update_url": ""
    },
    "players": [
        {
            "attributes": [
                "bot"
            ],
            "last_seen": {
                "player_name": "personman",
                "time": 1677390631
            },
            "steamid": "76561199006548700"
        },
        {
            "attributes": [
                "bot"
            ],
            "last_seen": {
                "player_name": "ุ",
                "time": 1678491214
            },
            "steamid": 76561198084134025
        },
        {
            "attributes": [
                "bot"
            ],
            "last_seen": {
                "player_name": "x",
                "time": 1677390631
            },
			"steamid": 76561199006548705}
    ]
}
`)

	good := fixSteamIDFormat(badSchema)
	require.Equal(t, goodSchema, good)
}
