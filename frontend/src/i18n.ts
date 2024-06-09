import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import LanguageDetector from 'i18next-browser-languagedetector';

const resources = {
    en: {
        common: {
            appname: 'Bot Detector',
            button: {
                save: 'Save',
                cancel: 'Cancel',
                reset: 'Reset',
                clear: 'Clear',
                ok: 'OK'
            },
            yes: 'yes',
            no: 'no',
            mark_new_tag: {
                title: 'Mark Player With New Tag',
                tag: 'New Tag'
            },
            new_kick_tag: {
                title: 'Create New Auto Kick Tag',
                tag: 'Tags'
            },
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
                    whitelist_label: 'Whitelist',
                    vote_label: 'Call Vote'
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
                    open_home: 'Go Home',
                    game_state_running: 'Game is currently running',
                    game_state_stopped: 'Launch TF2!'
                }
            },
            settings: {
                label: 'Settings',
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
                    auto_launch_game_label: 'Auto Launch Game',
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
                    description: 'Auto-Kicker, Tags, Chat Warnings'
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
                        'What address the http service will listen on. (The URL you are connected to right now). You should use localhost ' +
                        'unless you know what you are doing as there is no authentication system.',
                    http_notice:
                        '* Must restart application for changes to take effect.'
                },
                steam: {
                    label: 'Steam & API Config',
                    description:
                        'Configure Steam API, Game & Remote Data Sources',
                    steam_id_label: 'Steam ID',
                    steam_id_tooltip:
                        'You can choose one of the following formats: steam,steam3,steam64',
                    api_key_label: 'Steam API Key',
                    api_key_tooltip: 'Your personal steam web api key',
                    bd_api_enabled_label: 'Enable bd-api Integration',
                    bd_api_enabled_tooltip:
                        'Enabling bd-api integration will give access to several more data points about players.' +
                        'This includes information such as sourcebans history and, eventually, competitive history.' +
                        'Using it will cause all Steam API requests to also be proxied over the bd-api service' +
                        'automatically. Because of this, using bd-api removes the requirement to have a Steam API key set',
                    bd_api_address_label: 'Custom bd-api URL',
                    bd_api_address_tooltip:
                        'URL to a custom instance of bd-api',
                    steam_dir_label: 'Steam Root Directory',
                    steam_dir_tooltip:
                        'Location of your steam installation directory containing your userdata folder'
                },
                tf2: {
                    label: 'TF2 Config',
                    description: 'Configure game settings',
                    tf2_dir_label: 'TF2 Root Directory',
                    tf2_dir_tooltip:
                        'Path to your `steamapps/common/Team Fortress 2/tf` Folder',
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
                    edit_title: `Edit List:`,
                    enabled_label: 'Enabled'
                }
            }
        }
    },
    ru: {
        common: {
            appname: 'Bot Detector',
            button: {
                save: 'Сохранить',
                cancel: 'Отмена',
                clear: 'Очистить'
            },
            yes: 'да',
            no: 'нет',
            player_table: {
                column: {
                    user_id: 'uid',
                    name: 'имя',
                    score: 'счёт',
                    kills: 'убийства',
                    deaths: 'смерти',
                    kpm: 'у/м',
                    health: 'здоровье',
                    connected: 'время',
                    map_time: 'время карты',
                    ping: 'пинг'
                },
                notes: {
                    title: 'Изменить Заметки О Игроке',
                    note_label: 'Заметка'
                },
                row: {
                    icon_dead: 'Игрок мёртв (хах)',
                    vac_bans: 'Известные VAC баны',
                    source_bans: 'Известные Sourcebans',
                    player_on_lists:
                        'Игрок отмечен в одном или нескольких списках',
                    player_on_lists_whitelisted:
                        'Игрок отмечен, но находится в белом списке',
                    player_notes: 'Есть заметки о игроке'
                },
                menu: {
                    mark_label: 'Отметить Игрока Как',
                    external_label: 'Открыть Внкшнюю Ссылку',
                    copy_label: 'Скопировать SteamID',
                    chat_history_label: 'История Чата',
                    name_history_label: 'История Имён',
                    remove_whitelist_label: 'Удалить Из Белого Списка',
                    whitelist_label: 'Добавить В Белый Список'
                },
                details: {
                    uid_label: 'UID',
                    name_label: 'Имя',
                    visibility_label: 'Видимость Профиля',
                    vac_bans_label: 'Vac Баны',
                    game_bans_label: 'Игровые Баны',
                    matches: {
                        origin_label: 'Источник',
                        type_label: 'Тип',
                        tags_label: 'Метка'
                    },
                    sourcebans: {
                        site_name_label: 'Имя Сайта',
                        created_label: 'Создан',
                        perm_label: 'Постоянный',
                        reason_label: 'Причина'
                    }
                }
            },
            toolbar: {
                button: {
                    show_only_negative:
                        'Отображать только игроков с каким-либо негативным статусом',
                    shown_columns: 'Настроить отображаемые столбцы',
                    open_settings: 'Открыть настройки',
                    game_state_running: 'Игра запущена',
                    game_state_stopped: 'Запустить TF2!'
                }
            },
            settings: {
                general: {
                    label: 'Общие',
                    description: 'Kicker, Метки, Предупреждения В Чате',
                    chat_warnings_label: 'Предупреждения В Чате',
                    chat_warnings_tooltip:
                        'Активировать предупреждения в игровом чате в текущей игре',
                    kicker_enabled_label: 'Kicker Активирован',
                    kicker_enabled_tooltip:
                        'Включить функционал автоматического начала голосования',
                    kick_tags_label: 'Выгоняемые метки',
                    kick_tags_tooltip:
                        'Только при совпадениях по этим меткам сработает начало голосования или предупреждение.',
                    party_warnings_enabled_label:
                        'Предупреждения Лобби Активированы',
                    party_warnings_enabled_tooltip:
                        'Активировать отправку лог сообщений в чат лобби',
                    discord_presence_enabled_label:
                        'Discord Presence Активирован',
                    discord_presence_enabled_tooltip:
                        'Активировать отображение игрового статуса в твоём дискорд профиле.',
                    auto_launch_game_label: 'Авто Запуск Игры',
                    auto_launch_game_tooltip:
                        'Когда активирован, при запуске bd, TF2 так же будет запущена',
                    auto_close_on_game_exit_label:
                        'Авто Закрытие При Выходе Из Игры',
                    auto_close_on_game_exit_tooltip:
                        'Когда активирован, при выходе из игры, так же закрывается bd.',
                    debug_log_enabled_label: 'Активирован Дебаг Лог',
                    debug_log_enabled_tooltip:
                        'Когда активирован, логи записываются в bd.log в папку конфигурации приложения'
                },
                player_lists: {
                    label: 'Списки Игроков и Правил',
                    description: 'Авто-Kicker, метки, Предупреждения в чате'
                },
                external_links: {
                    label: 'Внешние Ссылки',
                    description: 'Настроить меню кастомных ссылок'
                },
                http: {
                    label: 'HTTP Сервис',
                    description: 'Базовые настройки HTTP сервиса',
                    http_enabled_label: 'Активировать HTTP Сервис*',
                    http_enabled_tooltip:
                        'ВНИМАНИЕ: HTTP сервис необходим для функционирования виджета браузера (этой страницы). Его можно активировать ' +
                        'только вручную изменив файл конфигурации',
                    http_listen_addr_label: 'Прослушиваемый Адрес (host:port)',
                    http_listen_addr_tooltip:
                        'Какой адрес http сервис будет прослушивать. (Ссылка по которой вы сейчас подключены). Вам стоит использовать localhost ' +
                        'если вы не знаете что делаете, так как системы аутенфикации нет.'
                },
                steam: {
                    label: 'Конфигурация Steam',
                    description: 'Настройка steam api и клиента',
                    steam_id_label: 'Steam ID',
                    steam_id_tooltip:
                        'Вы можете выбрать один из следующих форматов: steam,steam3,steam64',
                    api_key_label: 'Steam API Key',
                    api_key_tooltip: 'Ваш личный steam web api key',
                    steam_dir_label: 'Папка Steam',
                    steam_dir_tooltip:
                        'Расположение вашей папки steam содержащую папку userdata'
                },
                tf2: {
                    label: 'Конфигурация TF2',
                    description: 'Изменить настройки игры',
                    tf2_dir_label: 'Папка TF2',
                    tf2_dir_tooltip:
                        'Путь к вашей `steamapps/common/Team Fortress 2/tf` Папке',
                    rcon_static_label: 'Статичный Режим RCON',
                    rcon_static_tooltip:
                        'Когда активирован, rcon будет всегда использовать статичные порт и пароль - 21212 / pazer_sux_lol. Иначе они будут рандомно сгенерированы при запуске игры',
                    voice_bans_enabled_label: 'Сгенерировать Голосовые Баны',
                    voice_bans_enabled_tooltip:
                        'ВНИМАНИЕ: Это перепишет ваш текущий список банов. Заглушает 200 самых недавних помеченных игроков.'
                },
                link_editor: {
                    create_title: `Создать Новую Ссылку`,
                    edit_title: `Редактировать Ссылку: `,
                    enabled_label: 'Активирован',
                    steam_id_format: 'Формат Steam ID'
                },
                list_editor: {
                    create_title: `Создать Новый Список`,
                    edit_title: `Редактировать Список:`
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
