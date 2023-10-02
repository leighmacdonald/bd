import React, { useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { IconMenuItem, NestedMenuItem } from 'mui-nested-menu';
import ArrowRightOutlinedIcon from '@mui/icons-material/ArrowRightOutlined';
import LinkOutlinedIcon from '@mui/icons-material/LinkOutlined';
import FlagIcon from '@mui/icons-material/Flag';
import { formatExternalLink, logError, openInNewTab } from '../../util';
import { Link } from '../../api';
import { SteamIDProps, SubMenuProps } from './common';

interface LinksMenuProps {
    links: Link[];
}

export const LinksMenu = ({
    contextMenuPos,
    links,
    steam_id,
    onClose
}: LinksMenuProps & SteamIDProps & SubMenuProps) => {
    const { t } = useTranslation();

    const onClickLink = useCallback((steam_id: string, link: Link) => {
        try {
            openInNewTab(formatExternalLink(steam_id, link));
        } catch (e) {
            logError(e);
        } finally {
            onClose();
        }
    }, []);

    return (
        <NestedMenuItem
            rightIcon={<ArrowRightOutlinedIcon />}
            leftIcon={<LinkOutlinedIcon color={'primary'} />}
            label={t('player_table.menu.external_label')}
            parentMenuOpen={contextMenuPos !== null}
        >
            {links.map((link) => (
                <IconMenuItem
                    leftIcon={<FlagIcon color={'primary'} />}
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
