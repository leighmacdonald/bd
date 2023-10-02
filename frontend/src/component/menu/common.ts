export interface SubMenuProps {
    contextMenuPos: NullablePosition;
    onClose: () => void;
}

export interface SteamIDProps {
    steam_id: string;
}

export type NullablePosition = {
    mouseX: number;
    mouseY: number;
} | null;
