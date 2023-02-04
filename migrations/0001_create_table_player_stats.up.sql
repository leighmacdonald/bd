create table if not exists player
(
    steam_id integer primary key,
    kills_on int not null default 0,
    deaths_by int not null default 0,
    created_on date not null default (DATETIME('now')),
    updated_on date not null default (DATETIME('now'))
);

create table if not exists player_names
(
    name_id int primary key,
    steam_id integer not null REFERENCES player(steam_id),
    name text not null,
    created_on date not null default (DATETIME('now'))
);

create index if not exists idx_steam_id_name on player_names (steam_id, name);

create table if not exists player_messages (
    message_id int primary key,
    steam_id integer not null REFERENCES player(steam_id),
    message text not null,
    created_on date not null default (DATETIME('now'))
);
