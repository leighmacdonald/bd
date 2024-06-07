create table if not exists config (
    steam_id                  text    not null default '',
    steam_dir                 text    not null default '',
    tf2_dir                   text    not null default '',
    auto_launch_game          boolean not null default false,
    auto_close_on_game_exit   boolean not null default false,
    bd_api_enabled            boolean not null default true,
    bd_api_address            text    not null default 'https://bd-api.roto.lol',
    api_key                   text    not null default '',
    systray_enabled           boolean not null default true,
    disconnected_timeout      integer not null default 60 check ( disconnected_timeout >= 0 ),
    discord_presence_enabled  boolean not null default false,
    kicker_enabled            boolean not null default false,
    chat_warnings_enabled     boolean not null default true,
    voice_bans_enabled        boolean not null default false,
    debug_log_enabled         boolean not null default false,
    rcon_static               boolean not null default false,
    http_enabled              boolean not null default true,
    http_listen_addr          text    not null default 'localhost:8900',
    player_expired_timeout    int     not null default 10 check ( player_expired_timeout >= 0 ),
    player_disconnect_timeout int     not null default 60 check ( player_disconnect_timeout >= 0 ),
    run_mode                  text    not null default 'release' CHECK ( run_mode IN ('release', 'dev', 'test') ),
    log_level                 text    not null default 'error' CHECK ( log_level IN ('error', 'debug', 'warning', 'info') ),
    rcon_address              text    not null default '127.0.0.1',
    rcon_port                 int     not null default 51944 CHECK ( rcon_port > 0 AND rcon_port < 65535 ),
    rcon_password             text    not null default (lower(hex(randomblob(8))))
);

-- Create default row
insert into config (steam_id)
values (0);

create table if not exists links
(
    link_id    integer primary key,
    name       text unique not null,
    url        text unique not null,
    id_format  text        not null default 'steam64' CHECK ( id_format IN ('steam64', 'steam3', 'steam') ),
    enabled    boolean     not null default true,
    created_on date        not null default (DATETIME('now')),
    updated_on date        not null default (DATETIME('now'))
);

insert into links (name, url, id_format, enabled)
VALUES ('RGL', 'https://rgl.gg/Public/PlayerProfile.aspx?p=%d', 'steam64', true),
       ('Steam', 'https://steamcommunity.com/profiles/%d', 'steam64', true),
       ('OzFortress', 'https://ozfortress.com/users/steam_id/%d', 'steam64', true),
       ('ESEA', 'https://play.esea.net/index.php?s=search&query=%s', 'steam3', true),
       ('UGC', 'https://www.ugcleague.com/players_page.cfm?player_id=%d', 'steam64', true),
       ('ETF2L', 'https://etf2l.org/search/%d/', 'steam64', true),
       ('trends.tf', 'https://trends.tf/player/%d/', 'steam64', true),
       ('demos.tf', 'https://demos.tf/profiles/%d', 'steam64', true),
       ('logs.tf', 'https://logs.tf/profile/%d', 'steam64', true);

alter table lists
    add column name text not null default '';

insert into lists (list_type, name, url, enabled)
VALUES (1, '@trusted', 'https://trusted.roto.lol/v1/steamids', false),
       (1, 'Pazer',
        'https://raw.githubusercontent.com/PazerOP/tf2_bot_detector/master/staging/cfg/playerlist.official.json',
        false),
       (1, 'Uncletopia', 'https://uncletopia.com/export/bans/tf2bd', false)
;
