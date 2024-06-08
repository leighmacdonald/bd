const baseUrl = `${location.protocol}//${location.host}`;
const headers: Record<string, string> = {
    'Content-Type': 'application/json; charset=UTF-8'
};

const call = async <TRequest = emptyBody>(
    method: string,
    path: string,
    body?: TRequest
) => {
    const opts: RequestInit = {
        mode: 'same-origin',
        method: method.toUpperCase()
    };
    if (method !== 'GET' && body) {
        opts['body'] = JSON.stringify(body);
    }
    opts.headers = headers;
    const url = new URL(path, baseUrl);
    return await fetch(url, opts);
};

const callJson = async <TResponse, TRequest = emptyBody>(
    method: string,
    path: string,
    body?: TRequest
) => {
    const resp = await call<TRequest>(method, path, body);
    return (await resp.json()) as TResponse;
};

export type emptyBody = object;

export type steamIdFormat = 'steam64' | 'steam3' | 'steam32' | 'steam';

enum ProfileVisibility {
    ProfileVisibilityPrivate = 1,
    ProfileVisibilityFriendsOnly = 2,
    ProfileVisibilityPublic = 3
}

export const visibilityString = (v: ProfileVisibility): string => {
    switch (v) {
        case ProfileVisibility.ProfileVisibilityPublic:
            return 'Public';
        case ProfileVisibility.ProfileVisibilityFriendsOnly:
            return 'Friends Only';
        default:
            return 'Private';
    }
};

export enum Team {
    SPEC,
    UNASSIGNED,
    BLU,
    RED
}

export interface Match {
    origin: string;
    attributes: string[];
    matcher_type: string;
}

export interface Server {
    server_name: string;
    current_map: string;
    tags: string[];
    last_update: string;
}

export interface GameState {
    game_running: boolean;
    server: Server;
    players: Player[];
}

const defaultSteamAvatarHash = 'fef49e7fa7e1997310d705b2a6158ff8dc1cdfeb';

export const avatarURL = (hash: string, size = 'full'): string =>
    `https://avatars.cloudflare.steamstatic.com/${
        hash != '' ? hash : defaultSteamAvatarHash
    }_${size}.jpg`;

export interface Player {
    steam_id: string;
    personaname: string;
    visibility: ProfileVisibility;
    real_name: string;
    account_created_on: Date;
    avatar_hash: string;
    community_banned: boolean;
    game_bans: number;
    vac_bans: number;
    last_vac_ban_on: number;
    kills_on: number;
    deaths_by: number;
    rage_quits: number;
    notes: string;
    whitelist: boolean;
    profile_updated_on: Date;
    created_on: Date;
    updated_on: Date;
    economy_ban: boolean;
    team: Team;
    connected: number;
    map_time: number;
    user_id: number;
    ping: number;
    score: number;
    is_connected: boolean;
    alive: boolean;
    health: number;
    valid: boolean;
    deaths: number;
    kills: number;
    kpm: number;
    kick_attempt_count: number;
    our_friend: boolean;
    sourcebans: SourcebansRecord[];
    matches: Match[];
}

export interface SourcebansRecord {
    ban_id: number;
    site_name: string;
    site_id: number;
    persona_name: string;
    steam_id: string;
    reason: string;
    duration: number;
    permanent: boolean;
    created_on: string;
}

export const formatSeconds = (seconds: number): string => {
    const h = Math.floor(seconds / 3600);
    const m = Math.floor((seconds % 3600) / 60);
    const s = Math.round(seconds % 60);
    return [h, m > 9 ? m : h ? '0' + m : m || '0', s > 9 ? s : '0' + s]
        .filter(Boolean)
        .join(':');
};

export interface List {
    list_type: string;
    name: string;
    enabled: boolean;
    url: string;
}

export interface Link {
    enabled: boolean;
    name: string;
    url: string;
    id_format: steamIdFormat;
    deleted: boolean;
}

export interface UserSettings {
    steam_id: string;
    steam_dir: string;
    tf2_dir: string;
    auto_launch_game: boolean;
    auto_close_on_game_exit: boolean;
    api_key: string;
    bd_api_enabled: boolean;
    bd_api_address: string;
    disconnected_timeout: string;
    discord_presence_enabled: boolean;
    kicker_enabled: boolean;
    chat_warnings_enabled: boolean;
    party_warnings_enabled: boolean;
    kick_tags: string[];
    voice_bans_enabled: boolean;
    debug_log_enabled: boolean;
    lists: List[];
    links: Link[];
    rcon_static: boolean;
    gui_enabled: boolean;
    http_enabled: boolean;
    http_listen_addr: string;
    player_expired_timeout: number;
    player_disconnect_timeout: number;
    unique_tags: string[];
}

export interface UserNote {
    note: string;
}

export type kickReasons = 'idle' | 'scamming' | 'cheating' | 'other';

const callVote = async (steamID: string, reason: kickReasons = 'cheating') => {
    await call('POST', `/api/callvote/${steamID}/${reason}`);
};

export const callVoteMutation = (steam_id: string) => {
    return {
        mutationKey: ['callVote', { steam_id }],
        mutationFn: async (vars: { reason: kickReasons }) => {
            return await callVote(steam_id, vars.reason);
        }
    };
};

const addWhitelist = async (steamId: string) =>
    await call('POST', `/api/whitelist/${steamId}`);

export const addWhitelistMutation = () => {
    return {
        mutationKey: ['addWhitelist'],
        mutationFn: async (variables: { steamId: string }) => {
            return await addWhitelist(variables.steamId);
        }
    };
};

const deleteWhitelist = async (steamId: string) =>
    await call('DELETE', `/api/whitelist/${steamId}`);

export const deleteWhitelistMutation = () => {
    return {
        mutationKey: ['deleteWhitelist'],
        mutationFn: async (variables: { steamId: string }) => {
            return await deleteWhitelist(variables.steamId);
        }
    };
};

const saveUserNote = async (steamId: string, notes: string) =>
    await call<UserNote>('POST', `/api/notes/${steamId}`, { note: notes });

export const saveUserNoteMutation = (steamId: string) => {
    return {
        mutationKey: ['saveUserNote', { steamId }],
        mutationFn: async (variables: { notes: string }) => {
            return await saveUserNote(steamId, variables.notes);
        }
    };
};

const deleteUserNote = async (steamId: string) =>
    await call<UserNote>('DELETE', `/api/notes/${steamId}`);

export const deleteUserNoteMutation = (steamId: string) => {
    return {
        mutationKey: ['deleteUserNote', { steamId }],
        mutationFn: async () => {
            return await deleteUserNote(steamId);
        }
    };
};

const markUser = async (steamId: string, attrs: string[]) =>
    await call('POST', `/api/mark/${steamId}`, { attrs });

export const markUserMutation = (steamId: string) => {
    return {
        mutationKey: ['markUser', { steamId }],
        mutationFn: async (vars: { attrs: string[] }) => {
            return await markUser(steamId, vars.attrs);
        }
    };
};

const unmarkUser = async (steamId: string) =>
    await call('DELETE', `/api/mark/${steamId}`);

export const unmarkUserMutation = (steamId: string) => {
    return {
        mutationKey: ['unmarkUser', { steamId }],
        mutationFn: async () => {
            return await unmarkUser(steamId);
        }
    };
};

const getState = async () => await callJson<GameState>('GET', '/api/state');

export const getStateOptions = () => {
    return {
        queryKey: ['state'],
        queryFn: getState,
        refetchInterval: 1000
    };
};

const getLaunch = async () => await callJson('GET', '/api/launch');

export const getLaunchOptions = () => {
    return {
        queryKey: ['launch'],
        queryFn: getLaunch
    };
};

const getQuit = async () => await callJson('GET', '/api/quit');

export const getQuitOptions = () => {
    return {
        queryKey: ['quit'],
        queryFn: getQuit
    };
};

const getUserSettings = async () =>
    await callJson<UserSettings>('GET', '/api/settings');

export const getUserSettingsQuery = () => {
    return {
        queryKey: ['userSettings'],
        queryFn: async () => {
            return await getUserSettings();
        }
    };
};

const saveUserSettings = async (settings: UserSettings) =>
    await call('PUT', '/api/settings', settings);

export const saveUserSettingsMutation = () => {
    return {
        mutationKey: ['userSettings'],
        mutationFn: async (variables: UserSettings) => {
            return await saveUserSettings(variables);
        }
    };
};
