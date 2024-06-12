import { Link } from './api';
import SteamID from 'steamid';
import { formatDuration } from 'date-fns/formatDuration';
import { Duration } from 'date-fns';

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

// eslint-disable-next-line @typescript-eslint/no-empty-function
export const noop = (): void => {};

/**
 * Get case insensitive unique string values
 * @param values
 */
export const uniqCI = (values: string[]): string[] => [
    ...new Map(values.map((s) => [s.toLowerCase(), s])).values()
];

const zeroPad = (num: number) => String(num).padStart(2, '0');

export const humanDuration = (duration: Duration) => {
    return formatDuration(duration, {
        format: ['hours', 'minutes', 'seconds'],
        zero: true,
        delimiter: ':',
        locale: {
            formatDistance: (_token, count) => zeroPad(count)
        }
    });
};
