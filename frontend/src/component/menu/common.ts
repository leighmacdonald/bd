export interface SubMenuProps {
    contextMenuPos: NullablePosition;
    onClose: () => void;
}

export interface SteamIDProps {
    steamId: string;
}

export type NullablePosition = {
    mouseX: number;
    mouseY: number;
} | null;
