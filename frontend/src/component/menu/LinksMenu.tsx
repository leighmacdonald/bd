import React, { useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { IconMenuItem, NestedMenuItem } from 'mui-nested-menu';
import ArrowRightOutlinedIcon from '@mui/icons-material/ArrowRightOutlined';
import LinkOutlinedIcon from '@mui/icons-material/LinkOutlined';
import { formatExternalLink, logError, openInNewTab } from '../../util';
import { Link, useUserSettings } from '../../api';
import { SteamIDProps, SubMenuProps } from './common';

export const LinksMenu = ({
    contextMenuPos,
    steam_id,
    onClose
}: SteamIDProps & SubMenuProps) => {
    const { t } = useTranslation();
    const { settings } = useUserSettings();
    const onClickLink = useCallback(
        (steam_id: string, link: Link) => {
            try {
                openInNewTab(formatExternalLink(steam_id, link));
            } catch (e) {
                logError(e);
            } finally {
                onClose();
            }
        },
        [onClose]
    );

    return (
        <NestedMenuItem
            rightIcon={<ArrowRightOutlinedIcon />}
            leftIcon={<LinkOutlinedIcon color={'primary'} />}
            label={t('player_table.menu.external_label')}
            parentMenuOpen={contextMenuPos !== null}
        >
            {settings.links
                .filter((link) => link.enabled)
                .map((link) => (
                    <IconMenuItem
                        onClick={() => {
                            onClickLink(steam_id, link);
                        }}
                        label={link.name}
                        key={`link-${steam_id}-${link.name}`}
                    />
                ))}
        </NestedMenuItem>
    );
};
