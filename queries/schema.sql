CREATE TABLE player
(
    steam_id           integer unique,
    visibility         integer not null default 3 check ( visibility >= 1 AND visibility <= 3 ),
    real_name          text    not null default '',
    account_created_on date    not null default 0,
    avatar_hash        text    not null default '',
    community_banned   integer not null default 0,
    game_bans          integer not null default 0,
    vac_bans           integer not null default 0,
    last_vac_ban_on    date,
    kills_on           integer not null default 0,
    deaths_by          integer not null default 0,
    rage_quits         integer not null default 0,
    notes              text    not null default '',
    whitelist          boolean not null default false,
    profile_updated_on date    not null default (DATETIME('now')),
    created_on         date    not null default (DATETIME('now')),
    updated_on         date    not null default (DATETIME('now'))
);

CREATE TABLE player_messages
(
    message_id integer primary key,
    steam_id   integer not null,
    message    text    not null,
    created_on date    not null default (DATETIME('now')),
    foreign key (steam_id) references player (steam_id) on delete cascade
);

CREATE TABLE player_names
(
    name_id    integer primary key,
    steam_id   integer not null,
    name       text    not null,
    created_on date    not null default (DATETIME('now')),
    foreign key (steam_id) references player (steam_id) on delete cascade
);


