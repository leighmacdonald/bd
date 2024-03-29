-- name: Player :one
SELECT p.steam_id,
       p.visibility,
       p.real_name,
       p.account_created_on,
       p.avatar_hash,
       p.community_banned,
       p.game_bans,
       p.vac_bans,
       p.last_vac_ban_on,
       p.kills_on,
       p.deaths_by,
       p.rage_quits,
       p.notes,
       p.whitelist,
       p.created_on,
       p.updated_on,
       p.profile_updated_on,
       p.personaname
FROM player p
WHERE p.steam_id = @steam_id;

-- name: PlayerInsert :one
INSERT INTO player (steam_id, personaname, visibility, real_name, account_created_on,
                    avatar_hash, community_banned, game_bans, vac_bans, last_vac_ban_on,
                    kills_on, deaths_by, rage_quits, notes, whitelist, profile_updated_on,
                    created_on, updated_on)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: PlayerUpdate :exec
UPDATE player
SET visibility         = @visibility,
    real_name          = @real_name,
    account_created_on = @account_created_on,
    avatar_hash        = @avatar_hash,
    community_banned   = @community_banned,
    game_bans          = @game_bans,
    vac_bans           = @vac_bans,
    last_vac_ban_on    = @last_vac_ban_on,
    kills_on           = @kills_on,
    deaths_by          = @deaths_by,
    rage_quits         = @rage_quits,
    notes              = @notes,
    whitelist          = @whitelist,
    updated_on         = @updated_on,
    profile_updated_on = @profile_updated_on,
    personaname        = @personaname
WHERE steam_id = @steam_id;

-- name: PlayerSearch :many
SELECT p.steam_id,
       p.visibility,
       p.real_name,
       p.account_created_on,
       p.avatar_hash,
       p.community_banned,
       p.game_bans,
       p.vac_bans,
       p.last_vac_ban_on,
       p.kills_on,
       p.deaths_by,
       p.rage_quits,
       p.notes,
       p.whitelist,
       p.profile_updated_on,
       p.created_on,
       p.updated_on,
       p.personaname
FROM player p
WHERE (@steam_id = 0 OR p.steam_id = @steam_id)
  AND (@name IS '' OR p.personaname LIKE @name)
ORDER BY p.updated_on DESC
LIMIT 1000;

-- name: UserNameSave :exec
INSERT INTO player_names (name_id, steam_id, name, created_on)
VALUES (?, ?, ?, ?);

-- name: UserNames :many
SELECT name_id, steam_id, name, created_on
FROM player_names
WHERE steam_id = @steam_id;

-- name: MessageSave :exec
INSERT INTO player_messages (message_id, steam_id, message, team, dead, created_on)
VALUES (?, ?, ?, ?, ?, ?);

-- name: Messages :many
SELECT message_id, steam_id, message, team, dead, created_on
FROM player_messages
WHERE steam_id = @steam_id;