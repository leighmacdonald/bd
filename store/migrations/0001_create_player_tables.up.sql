create table if not exists player
(
    steam_id           integer unique not null,
    personaname        text           not null default '',
    visibility         integer        not null default 3 check ( visibility >= 0 AND visibility <= 3 ),
    real_name          text           not null default '',
    account_created_on date           not null default 0,
    avatar_hash        text           not null default '',
    community_banned   boolean        not null default false,
    game_bans          integer        not null default 0,
    vac_bans           integer        not null default 0,
    last_vac_ban_on    date,
    kills_on           integer        not null default 0,
    deaths_by          integer        not null default 0,
    rage_quits         integer        not null default 0,
    notes              text           not null default '',
    whitelist          boolean        not null default false,
    profile_updated_on date           not null default (DATETIME('now')),
    created_on         date           not null default (DATETIME('now')),
    updated_on         date           not null default (DATETIME('now'))
);

create table if not exists player_names
(
    name_id    integer primary key,
    steam_id   integer not null,
    name       text    not null,
    created_on date    not null default (DATETIME('now')),
    foreign key (steam_id) references player (steam_id) on delete cascade
);

create index if not exists idx_player_name on player_names (steam_id, name);
create index if not exists idx_player_name_created_on on player_names (created_on);

create table if not exists player_messages
(
    message_id integer primary key,
    steam_id   integer not null,
    message    text    not null,
    team       boolean not null default false,
    dead       boolean not null default false,
    created_on date    not null default (DATETIME('now')),
    foreign key (steam_id) references player (steam_id) on delete cascade
);

create table if not exists player_friends
(
    steam_id        integer not null,
    steam_id_friend integer not null,
    friend_since    date    not null,
    created_on      date    not null default (DATETIME('now')),
    foreign key (steam_id) references player (steam_id) on delete cascade,
    primary key (steam_id, steam_id_friend)
);

create table if not exists player_sourcebans
(
    sourcebans_id integer primary key,
    steam_id      integer not null,
    site          text    not null,
    player_name   text    not null default '',
    reason        text    not null unique,
    duration      integer not null default 0,
    permanent     boolean not null default false,
    created_on    date    not null default (DATETIME('now')),
    foreign key (steam_id) references player (steam_id) on delete cascade
);

create table if not exists lists
(
    list_id    integer primary key,
    list_type  integer not null,
    url        text    not null unique,
    enabled    boolean not null default true,
    updated_on date    not null default (DATETIME('now')),
    created_on date    not null default (DATETIME('now'))
);
