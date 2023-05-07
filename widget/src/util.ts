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
        console.error(`Unable to write to clipboard: ${e}`);
    }
};

export const openInNewTab = (url: string) => {
    window.open(url, '_blank');
};

export const formatExternalLink = (steam_id: bigint, link: Link): string => {
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
