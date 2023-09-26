import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import LanguageDetector from 'i18next-browser-languagedetector';

const resources = {
    en: {
        common: {
            appname: 'Bot Detector',
            button: {
                save: 'Save',
                cancel: 'Cancel'
            },
            yes: 'yes',
            no: 'no',
            player_table: {
                column: {
                    user_id: 'uid',
                    name: 'name',
                    score: 'score',
                    kills: 'kills',
                    deaths: 'deaths',
                    kpm: 'kpm',
                    health: 'health',
                    connected: 'time',
                    map_time: 'map time',
                    ping: 'ping'
                },
                notes: {
                    title: 'Edit Player Notes',
                    note_label: 'Note'
                },
                row: {
                    icon_dead: 'Player is dead (lol)',
                    vac_bans: 'VAC bans on record',
                    source_bans: 'Sourcebans entries on record',
                    player_on_lists: 'Player is marked on one or more lists',
                    player_on_lists_whitelisted:
                        'Player is marked, but whitelisted',
                    player_notes: 'Player has notes'
                },
                menu: {
                    mark_label: 'Mark Player As',
                    external_label: 'Open External Link',
                    copy_label: 'Copy SteamID',
                    chat_history_label: 'Chat History',
                    name_history_label: 'Name History',
                    remove_whitelist_label: 'Remove Whitelist',
                    whitelist_label: 'Whitelist'
                },
                details: {
                    uid_label: 'UID',
                    name_label: 'Name',
                    visibility_label: 'Profile Visibility',
                    vac_bans_label: 'Vac Bans',
                    game_bans_label: 'Game Bans',
                    matches: {
                        origin_label: 'Origin',
                        type_label: 'Type',
                        tags_label: 'Tags'
                    },
                    sourcebans: {
                        site_name_label: 'Site Name',
                        created_label: 'Created',
                        perm_label: 'Perm',
                        reason_label: 'Reason'
                    }
                }
            },
            toolbar: {
                button: {
                    show_only_negative:
                        'Show only players with some sort of negative status',
                    shown_columns: 'Configure which columns are shown',
                    open_settings: 'Open the settings dialogue',
                    game_state_running: 'Game is currently running',
                    game_state_stopped: 'Launch TF2!'
                }
            },
            settings: {
                general: {
                    label: 'General',
                    description: 'Kicker, Tags, Chat Warnings',
                    chat_warnings_label: 'Chat Warnings',
                    chat_warnings_tooltip:
                        'Enable in-game chat warnings to be broadcast to the active game',
                    kicker_enabled_label: 'Kicker Enabled',
                    kicker_enabled_tooltip:
                        'Enable the bot auto kick functionality when a match is found',
                    kick_tags_label: 'Kickable Tag Matches',
                    kick_tags_tooltip:
                        'Only matches which also match these tags will trigger a kick or notification.',
                    party_warnings_enabled_label: 'Party Warnings Enabled',
                    party_warnings_enabled_tooltip:
                        'Enable log messages to be broadcast to the lobby chat window',
                    discord_presence_enabled_label: 'Discord Presence Enabled',
                    discord_presence_enabled_tooltip:
                        'Enable game status presence updates to your local discord client.',
                    auto_launch_game_label: 'Kickable Tag Matches',
                    auto_launch_game_tooltip:
                        'When enabled, upon launching bd, TF2 will also be launched at the same time',
                    auto_close_on_game_exit_label: 'Auto Close On Game Exit',
                    auto_close_on_game_exit_tooltip:
                        'When enabled, upon the game existing, also shutdown bd.',
                    debug_log_enabled_label: 'Enabled Debug Log',
                    debug_log_enabled_tooltip:
                        'When enabled, logs are written to bd.log in the application config root'
                },
                player_lists: {
                    label: 'Player & Rules Lists',
                    description: 'Auth-Kicker, Tags, Chat Warnings'
                },
                external_links: {
                    label: 'External Links',
                    description: 'Configure custom menu links'
                },
                http: {
                    label: 'HTTP Service',
                    description: 'Basic HTTP service settings',
                    http_enabled_label: 'Enable The HTTP Service*',
                    http_enabled_tooltip:
                        'WARN: The HTTP service enabled the browser widget (this page) to function. You can only re-enable this ' +
                        'service by editing the config file manually',
                    http_listen_addr_label: 'Listen Address (host:port)',
                    http_listen_addr_tooltip:
                        'What address the http service will listen on. (The URL you are connected to right now). You should use localhost' +
                        'unless you know what you are doing as there is no authentication system.'
                },
                steam: {
                    label: 'Steam Config',
                    description: 'Configure steam api & client',
                    steam_id_label: 'Steam ID',
                    steam_id_tooltip:
                        'You can choose one of the following formats: steam,steam3,steam64',
                    api_key_label: 'Steam API Key',
                    api_key_tooltip: 'Your personal steam web api key',
                    steam_dir_label: 'Steam Root Directory',
                    steam_dir_tooltip:
                        'Location of your steam installation directory containing your userdata folder'
                },
                tf2: {
                    label: 'TF2 Config',
                    description: 'Configure game settings',
                    tf2_dir_label: 'TF2 Root Directory',
                    tf2_dir_tooltip:
                        'Path to your steamapps/common/Team Fortress 2/tf` Folder',
                    rcon_static_label: 'RCON Static Mode',
                    rcon_static_tooltip:
                        'When enabled, rcon will always use the static port and password of 21212 / pazer_sux_lol. Otherwise these are generated randomly on game launch',
                    voice_bans_enabled_label: 'Generate Voice Bans',
                    voice_bans_enabled_tooltip:
                        'WARN: This will overwrite your current ban list. Mutes the 200 most recent marked entries.'
                },
                link_editor: {
                    create_title: `Create New Link`,
                    edit_title: `Edit Link: `,
                    enabled_label: 'Enabled',
                    steam_id_format: 'Steam ID Format'
                },
                list_editor: {
                    create_title: `Create New List`,
                    edit_title: `Edit List:`
                }
            }
        }
    },
    ru: {
        common: {
            settings: {
                general: {
                    label: '1st Бармен'
                }
            }
        }
    }
};

i18n.use(LanguageDetector).use(initReactI18next).init({
    resources,
    fallbackLng: 'en',
    //lng: 'en',
    defaultNS: 'common'
});
