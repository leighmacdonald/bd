import { useCallback, useContext } from 'react';
import { useTranslation } from 'react-i18next';
import { IconMenuItem, NestedMenuItem } from 'mui-nested-menu';
import ArrowRightOutlinedIcon from '@mui/icons-material/ArrowRightOutlined';
import LinkOutlinedIcon from '@mui/icons-material/LinkOutlined';
import { formatExternalLink, logError, openInNewTab } from '../../util';
import { Link } from '../../api';
import { SteamIDProps, SubMenuProps } from './common';
import { SettingsContext } from '../../context/SettingsContext';

export const LinksMenu = ({
    contextMenuPos,
    steamId,
    onClose
}: SteamIDProps & SubMenuProps) => {
    const { t } = useTranslation();
    const { settings } = useContext(SettingsContext);
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
                            onClickLink(steamId, link);
                        }}
                        label={link.name}
                        key={`link-${steamId}-${link.name}`}
                    />
                ))}
        </NestedMenuItem>
    );
};

export default LinksMenu;
