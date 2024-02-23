import { useTranslation } from 'react-i18next';
import { useCallback } from 'react';
import { deleteWhitelist } from '../../api';
import { IconMenuItem } from 'mui-nested-menu';
import NotificationsPausedOutlinedIcon from '@mui/icons-material/NotificationsPausedOutlined';
import { SteamIDProps, SubMenuProps } from './common';
import { logError } from '../../util';

export const RemoveWhitelistMenu = ({
    steam_id,
    onClose
}: SteamIDProps & SubMenuProps) => {
    const { t } = useTranslation();

    const onDeleteWhitelist = useCallback(
        async (steam_id: string) => {
            try {
                await deleteWhitelist(steam_id);
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
            label={t('player_table.menu.remove_whitelist_label')}
            onClick={async () => {
                await onDeleteWhitelist(steam_id);
            }}
        />
    );
};
export default RemoveWhitelistMenu;
