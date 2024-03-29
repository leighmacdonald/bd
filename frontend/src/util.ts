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
