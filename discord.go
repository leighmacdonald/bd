package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/leighmacdonald/bd/discord/client"
)

type mapConfig struct {
	mappedName string
}

func discordAssetNameMap(mapName string) string {
	mapConfigs := map[string]mapConfig{
		"map_cp_5gorge":              {},
		"map_cp_badlands":            {},
		"map_cp_cloak":               {},
		"map_cp_coldfront":           {},
		"map_cp_degrootkeep":         {},
		"map_cp_dustbowl":            {},
		"map_cp_egypt":               {},
		"map_cp_fastlane":            {},
		"map_cp_foundry":             {},
		"map_cp_freight":             {},
		"map_cp_gorge":               {},
		"map_cp_gorge_event":         {},
		"map_cp_granary":             {},
		"map_cp_gravelpit":           {},
		"map_cp_gullywash":           {},
		"map_cp_junction":            {},
		"map_cp_manor_event":         {},
		"map_cp_mercenarypark":       {},
		"map_cp_metalworks":          {},
		"map_cp_mossrock":            {},
		"map_cp_mountainlab":         {},
		"map_cp_powerhouse":          {},
		"map_cp_process":             {},
		"map_cp_snakewater":          {},
		"map_cp_snowplow":            {},
		"map_cp_standin":             {},
		"map_cp_steel":               {},
		"map_cp_sunshine":            {},
		"map_cp_sunshine_event":      {},
		"map_cp_vanguard":            {},
		"map_cp_well":                {},
		"map_cp_yukon":               {},
		"map_ctf_2fort":              {},
		"map_ctf_2fort_invasion":     {},
		"map_ctf_doublecross":        {},
		"map_ctf_foundry":            {},
		"map_ctf_gorge":              {},
		"map_ctf_hellfire":           {},
		"map_ctf_landfall":           {},
		"map_ctf_sawmill":            {},
		"map_ctf_thundermountain":    {},
		"map_ctf_turbine":            {},
		"map_ctf_well":               {},
		"map_itemtest":               {},
		"map_koth_badlands":          {},
		"map_koth_bagel_event":       {},
		"map_koth_brazil":            {},
		"map_koth_harvest":           {},
		"map_koth_harvest_event":     {},
		"map_koth_highpass":          {},
		"map_koth_king":              {},
		"map_koth_lakeside":          {},
		"map_koth_lakeside_event":    {},
		"map_koth_lazarus":           {},
		"map_koth_maple_ridge_event": {},
		"map_koth_moonshine_event":   {},
		"map_koth_nucleus":           {},
		"map_koth_probed":            {},
		"map_koth_product":           {},
		"map_koth_sawmill":           {},
		"map_koth_slasher":           {},
		"map_koth_slaughter_event":   {},
		"map_koth_suijin":            {},
		"map_koth_viaduct":           {},
		"map_koth_viaduct_event":     {},
		"map_mvm_bigrock":            {},
		"map_mvm_coaltown":           {},
		"map_mvm_decoy":              {},
		"map_mvm_ghost_town":         {},
		"map_mvm_mannhattan":         {},
		"map_mvm_mannworks":          {},
		"map_mvm_rottenburg":         {},
		"map_pass_brickyard":         {},
		"map_pass_district":          {},
		"map_pass_timbertown":        {},
		"map_pd_cursed_cove_event":   {},
		"map_pd_monster_bash":        {},
		"map_pd_pit_of_death_event":  {},
		"map_pd_watergate":           {},
		"map_pl_badwater":            {},
		"map_pl_barnblitz":           {},
		"map_pl_borneo":              {},
		"map_pl_cactuscanyon":        {},
		"map_pl_enclosure":           {},
		"map_pl_goldrush":            {},
		"map_pl_hoodoo":              {},
		"map_pl_millstone_event":     {},
		"map_pl_pier":                {},
		"map_pl_precipice_event":     {},
		"map_pl_rumble_event":        {},
		"map_pl_snowycoast":          {},
		"map_pl_swiftwater":          {},
		"map_pl_thundermountain":     {},
		"map_pl_upward":              {},
		"map_plr_bananabay":          {},
		"map_plr_hightower":          {},
		"map_plr_hightower_event":    {},
		"map_plr_pipeline":           {},
		"map_rd_asteroid":            {},
		"map_sd_doomsday":            {},
		"map_sd_doomsday_event":      {},
		"map_tc_hydro":               {},
		"map_tr_dustbowl":            {},
		"map_tr_target":              {},
	}

	foundConfig, found := mapConfigs[fmt.Sprintf("map_%s", mapName)]
	if !found {
		foundConfig = mapConfig{mappedName: "cp_cloak"}
	}

	if foundConfig.mappedName != "" {
		mapName = foundConfig.mappedName
	}

	return mapName
}

type discordState struct {
	client   *client.Client
	state    *gameState
	settings *settingsManager
}

func newDiscordState(state *gameState, settings *settingsManager) *discordState {
	return &discordState{
		settings: settings,
		state:    state,
		client:   client.New(),
	}
}

// discordStateUpdater handles updating the discord presence data with the current game state. It uses the
// discord local IPC socket.
func (d discordState) start(ctx context.Context) {
	const discordAppID = "1076716221162082364"

	timer := time.NewTicker(time.Second * 10)
	isRunning := false

	for {
		select {
		case <-timer.C:
			settings := d.settings.Settings()

			if !settings.DiscordPresenceEnabled {
				if isRunning {
					// Logout of existing connection on settings change
					if errLogout := d.client.Logout(); errLogout != nil {
						slog.Error("Failed to logout of discord client", errAttr(errLogout))
					}

					isRunning = false
				}

				continue
			}

			if !isRunning {
				if errLogin := d.client.Login(discordAppID); errLogin != nil {
					slog.Debug("Failed to login to discord", errAttr(errLogin))

					continue
				}

				isRunning = true
			}

			if isRunning {
				if errUpdate := d.discordUpdateActivity(); errUpdate != nil {
					slog.Error("Failed to update discord activity", errAttr(errUpdate))

					isRunning = false
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

func (d discordState) discordUpdateActivity() error {
	buttons := []*client.Button{
		{
			Label: "GitHub",
			URL:   "https://github.com/leighmacdonald/bd",
		},
	}
	server := d.state.CurrentServerState()

	if !server.Addr.IsLinkLocalUnicast() /*SDR*/ && !server.Addr.IsPrivate() && server.Addr != nil && server.Port > 0 {
		u := fmt.Sprintf("steam://connect/%s:%d", server.Addr.String(), server.Port)
		buttons = append(buttons, &client.Button{
			Label: "Connect",
			URL:   u,
		})
	}

	currentMap := discordAssetNameMap(server.CurrentMap)
	state := "Offline"

	// TODO
	//if inGame {
	//	state = "In-Game"
	//}

	details := "Idle"
	if server.ServerName != "" {
		details = server.ServerName
	}

	playerCount := len(d.state.players.current())

	var party *client.Party
	if playerCount > 0 {
		// discord requires >=1
		party = &client.Party{
			Players:    playerCount,
			MaxPlayers: 24,
		}
	}

	if errSetActivity := d.client.SetActivity(client.Activity{
		State:      state,
		Details:    details,
		LargeImage: fmt.Sprintf("map_%s", currentMap),
		LargeText:  currentMap,
		SmallImage: "logo_cd",
		SmallText:  "",
		Party:      party,
		//Timestamps: &client.Timestamps{
		//	Start: &startupTime,
		//},
		Buttons: buttons,
	}); errSetActivity != nil {
		return errors.Join(errSetActivity, errDiscordActivity)
	}

	return nil
}
