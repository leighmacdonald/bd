create table if not exists player
(
    steam_id  integer primary key,
    kills_on int default 0,
    deaths_by int default 0,
    created_on date default (DATETIME('now')),
    updated_on date default (DATETIME('now'))
);

create table if not exists player_names
(
    name_id int primary key,
    steam_id integer,
    name text,
    created_on date default (DATETIME('now')),
    FOREIGN KEY(steam_id) REFERENCES player(steam_id)
);

create index if not exists idx_steam_id_name on player_names (steam_id, name);

create table if not exists player_messages (
    message_id int primary key,
    steam_id integer,
    message text,
    created_on date default (DATETIME('now')),
    FOREIGN KEY(steam_id) REFERENCES player(steam_id)
);

create index if not exists idx_steam_id_name on player_messages (steam_id, message);
