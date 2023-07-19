import { useEffect, useState } from 'react';
import { defaultUserSettings } from './context/settings';

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

export interface State {
    server: Server;
    players: Player[];
}

export interface Player {
    steam_id: string;
    name: string;
    created_on: Date;
    updated_on: Date;
    team: Team;
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
    valid: boolean;
    score: number;
    is_connected: boolean;
    kick_attempt_count: number;
    alive: boolean;
    deaths: number;
    health: number;
    our_friend: boolean;
    sourcebans: SourcebansRecord[] | null;
    matches: Match[] | null;
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

export const addWhitelist = async (steamId: string) =>
    await call('POST', `/whitelist/${steamId}`);

export const deleteWhitelist = async (steamId: string) =>
    await call('DELETE', `/whitelist/${steamId}`);

export const saveUserNote = async (steamId: string, notes: string) =>
    await call<UserNote>('POST', `/notes/${steamId}`, { note: notes });

export const deleteUserNote = async (steamId: string) =>
    await call<UserNote>('POST', `/notes/${steamId}`, { note: '' });

const getState = async () => await callJson<State>('GET', '/state');

const getUserSettings = async () =>
    await callJson<UserSettings>('GET', '/settings');

export const saveUserSettings = async (settings: UserSettings) => {
    await call('PUT', '/settings', settings);
};

export const useUserSettings = () => {
    const [settings, setSettings] = useState<UserSettings>(defaultUserSettings);
    const [error, setError] = useState<unknown>(null);
    const [loading, setLoading] = useState(false);

    useEffect(() => {
        setLoading(true);
        getUserSettings()
            .then((resp) => resp)
            .then(setSettings)
            .catch((e) => {
                console.log(e);
                setError(e);
            })
            .finally(() => setLoading(false));
    }, []);

    return { settings, error, loading };
};

export const useCurrentState = () => {
    const [state, setState] = useState<State>({
        server: {
            server_name: 'Unknown',
            current_map: 'Unknown',
            tags: [],
            last_update: ''
        },
        players: []
    });
    useEffect(() => {
        const interval = setInterval(async () => {
            try {
                setState(await getState());
            } catch (e) {
                console.log(e);
            }
        }, 1000);
        return () => {
            clearInterval(interval);
        };
    }, []);
    return state;
};
