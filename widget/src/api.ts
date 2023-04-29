import SteamID from 'steamid';

export const call = async <TRequest = null, TResponse = null>(
    method: string,
    path: string,
    body?: TRequest
) => {
    const headers: Record<string, string> = {
        'Content-Type': 'application/json; charset=UTF-8'
    };
    const opts: RequestInit = {
        mode: 'cors',
        credentials: 'include',
        method: method.toUpperCase()
    };
    if (method !== 'GET' && body) {
        opts['body'] = JSON.stringify(body);
    }
    opts.headers = headers;
    const url = new URL(
        `http://localhost:8900/${path}`,
        `${location.protocol}//${location.host}`
    );

    const resp = await fetch(url, opts);
    if (!resp.ok) {
        throw await resp.json();
    }
    const json: TResponse = await resp.json();
    return json;
};

enum ProfileVisibility {
    ProfileVisibilityPrivate = 1,
    ProfileVisibilityFriendsOnly = 2,
    ProfileVisibilityPublic = 3
}

export interface Player {
    steam_id: SteamID;
    name: string;
    created_on: Date;
    updated_on: Date;
    profile_updated_on: Date;
    kills_on: number;
    rage_quits: number;
    deaths_by: number;
    notes: string;
    whitelisted: boolean;
    real_name: string;
    name_previous: string;
    account_created_on: Date;
    visibility: ProfileVisibility;
    avatar_hash: string;
    community_banned: boolean;
    number_of_vac_bans: number;
    last_vac_ban_on: Date | null;
    number_of_game_bans: number;
    economy_ban: boolean;
    connected: number;
    user_id: number;
    ping: number;
    kills: number;
    deaths: number;
    our_friend: boolean;
    match: any;
}

export const getPlayers = async () => {
    return await call<null, Player[]>('GET', 'players');
};
