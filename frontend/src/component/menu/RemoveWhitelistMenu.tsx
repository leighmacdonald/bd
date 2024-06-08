import { useTranslation } from 'react-i18next';
import { useCallback } from 'react';
import { IconMenuItem } from 'mui-nested-menu';
import NotificationsPausedOutlinedIcon from '@mui/icons-material/NotificationsPausedOutlined';
import { SteamIDProps, SubMenuProps } from './common';
import { useMutation } from '@tanstack/react-query';
import { deleteWhitelistMutation } from '../../api.ts';

export const RemoveWhitelistMenu = ({
    steamId,
    onClose
}: SteamIDProps & SubMenuProps) => {
    const { t } = useTranslation();

    const mutation = useMutation(deleteWhitelistMutation());

    const onDeleteWhitelist = useCallback(
        async (steamId: string) => {
            mutation.mutate({ steamId });
        },

        [onClose]
    );

    return (
        <IconMenuItem
            leftIcon={<NotificationsPausedOutlinedIcon color={'primary'} />}
            label={t('player_table.menu.remove_whitelist_label')}
            onClick={async () => {
                await onDeleteWhitelist(steamId);
            }}
        />
    );
};
export default RemoveWhitelistMenu;
