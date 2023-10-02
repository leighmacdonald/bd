import React from 'react';
import { useTranslation } from 'react-i18next';
import Menu from '@mui/material/Menu';
import MenuItem from '@mui/material/MenuItem';
import { IconMenuItem } from 'mui-nested-menu';
import ForumOutlinedIcon from '@mui/icons-material/ForumOutlined';
import BadgeOutlinedIcon from '@mui/icons-material/BadgeOutlined';
import { MarkMenu } from './MarkMenu';
import { UnmarkMenu } from './UnmarkMenu';
import { LinksMenu } from './LinksMenu';
import { SteamIDMenu } from './SteamIDMenu';
import { RemoveWhitelistMenu } from './RemoveWhitelistMenu';
import { WhitelistMenu } from './WhitelistMenu';
import { NotesMenu } from './NotesMenu';
import { avatarURL, Player, UserSettings } from '../../api';
import { CallVoteMenu } from './CallVoteMenu';
import { SubMenuProps } from './common';

interface PlayerContextMenuProps {
    player: Player;
    settings: UserSettings;
}

/**
 * Context menu shown when right-clicking a player.
 *
 * @param contextMenuPos
 * @param player
 * @param settings
 * @param onClose
 * @constructor
 */
export const PlayerContextMenu = ({
    contextMenuPos,
    player,
    settings,
    onClose
}: PlayerContextMenuProps & SubMenuProps) => {
    const { t } = useTranslation();

    return (
        <Menu
            open={contextMenuPos !== null}
            onClose={onClose}
            anchorReference="anchorPosition"
            anchorPosition={
                contextMenuPos !== null
                    ? {
                          top: contextMenuPos.mouseY,
                          left: contextMenuPos.mouseX
                      }
                    : undefined
            }
        >
            <MenuItem disableRipple>
                <img alt={`Avatar`} src={avatarURL(player.avatar_hash)} />
            </MenuItem>
            <MarkMenu
                contextMenuPos={contextMenuPos}
                unique_tags={settings.unique_tags}
                steam_id={player.steam_id}
                onClose={onClose}
            />
            <UnmarkMenu
                steam_id={player.steam_id}
                contextMenuPos={contextMenuPos}
                onClose={onClose}
            />
            <CallVoteMenu
                contextMenuPos={contextMenuPos}
                steam_id={player.steam_id}
                onClose={onClose}
            />
            <LinksMenu
                contextMenuPos={contextMenuPos}
                links={settings.links}
                steam_id={player.steam_id}
                onClose={onClose}
            />
            <SteamIDMenu
                onClose={onClose}
                steam_id={player.steam_id}
                contextMenuPos={contextMenuPos}
            />
            <IconMenuItem
                leftIcon={<ForumOutlinedIcon color={'primary'} />}
                label={t('player_table.menu.chat_history_label')}
            />
            <IconMenuItem
                leftIcon={<BadgeOutlinedIcon color={'primary'} />}
                label={t('player_table.menu.name_history_label')}
            />
            {player.whitelisted ? (
                <RemoveWhitelistMenu
                    contextMenuPos={contextMenuPos}
                    steam_id={player.steam_id}
                    onClose={onClose}
                />
            ) : (
                <WhitelistMenu
                    steam_id={player.steam_id}
                    contextMenuPos={contextMenuPos}
                    onClose={onClose}
                />
            )}
            <NotesMenu
                notes={player.notes}
                steam_id={player.steam_id}
                onClose={onClose}
            />
        </Menu>
    );
};
