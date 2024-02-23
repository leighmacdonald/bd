import { Link } from './api';
import SteamID from 'steamid';

export const writeToClipboard = async (rawData: string) => {
    const data = [
        new ClipboardItem({
            'text/plain': new Blob([rawData], { type: 'text/plain' })
        })
    ];
    try {
        await navigator.clipboard.write(data);
    } catch (e) {
        logError(`Unable to write to clipboard: ${e}`);
    }
};

export const isStringIp = (value: string): boolean => {
    return /^(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$/.test(
        value
    );
};

export const openInNewTab = (url: string) => {
    window.open(url, '_blank');
};

export interface onClickProps {
    onClick: () => void;
}

export const formatExternalLink = (steam_id: string, link: Link): string => {
    const sid = new SteamID(steam_id);
    switch (link.id_format) {
        case 'steam':
            return link.url.replace('%s', sid.getSteam2RenderedID());
        case 'steam3':
            return link.url.replace('%s', sid.getSteam3RenderedID());
        case 'steam64':
            return link.url.replace('%d', steam_id.toString());
        default:
            return link.url;
    }
};

// TODO Send errors to backend for logging
// TODO Show error modal or notice somewhere
export const logError = (error: unknown) => {
    console.error(error);
};

export const isValidUrl = (urlString: string): boolean => {
    try {
        const newUrl = new URL(urlString);
        return (
            (newUrl.protocol === 'http:' || newUrl.protocol === 'https:') &&
            newUrl.host != ''
        );
    } catch (e) {
        return false;
    }
};

export const noop = (): void => {};

/**
 * Get case insensitive unique string values
 * @param values
 */
export const uniqCI = (values: string[]): string[] => [
    ...new Map(values.map((s) => [s.toLowerCase(), s])).values()
];

export const validatorSteamID = (value: string): string => {
    let err = 'Invalid SteamID';
    try {
        const id = new SteamID(value);
        if (id.isValid()) {
            err = '';
        }
    } catch (_) {
        /* empty */
    }
    return err;
};

export const makeValidatorLength = (length: number): inputValidator => {
    return (value: string): string => {
        if (value.length != length) {
            return 'Invalid value';
        }
        return '';
    };
};

export const validatorAddress = (value: string): string => {
    const pcs = value.split(':');
    if (pcs.length != 2) {
        return 'Format must match host:port';
    }
    if (pcs[0].toLowerCase() != 'localhost' && !isStringIp(pcs[0])) {
        return 'Invalid address. x.x.x.x or localhost accepted';
    }
    const port = parseInt(pcs[1], 10);
    if (!/^\d+$/.test(pcs[1])) {
        return 'Invalid port, must be positive integer';
    }
    if (port <= 0 || port > 65535) {
        return 'Invalid port, must be in range: 1-65535';
    }
    return '';
};

export type inputValidator = (value: string) => string | null;
