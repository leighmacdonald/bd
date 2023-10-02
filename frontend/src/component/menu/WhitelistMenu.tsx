import React, { useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { addWhitelist } from '../../api';
import { IconMenuItem } from 'mui-nested-menu';
import NotificationsPausedOutlinedIcon from '@mui/icons-material/NotificationsPausedOutlined';
import { SteamIDProps, SubMenuProps } from './common';
import { logError } from '../../util';

export const WhitelistMenu = ({
    steam_id,
    onClose
}: SteamIDProps & SubMenuProps) => {
    const { t } = useTranslation();

    const onAddWhitelist = useCallback(
        async (steamId: string) => {
            try {
                await addWhitelist(steamId);
            } catch (e) {
                logError(e);
            } finally {
                onClose();
            }
        },
        [onClose]
    );

    return (
        <IconMenuItem
            leftIcon={<NotificationsPausedOutlinedIcon color={'primary'} />}
            label={t('player_table.menu.whitelist_label')}
            onClick={async () => {
                await onAddWhitelist(steam_id);
            }}
        />
    );
};
