import { useTranslation } from 'react-i18next';
import { addWhitelistMutation } from '../../api';
import { IconMenuItem } from 'mui-nested-menu';
import NotificationsPausedOutlinedIcon from '@mui/icons-material/NotificationsPausedOutlined';
import { SteamIDProps, SubMenuProps } from './common';
import { logError } from '../../util';
import { useMutation } from '@tanstack/react-query';

export const WhitelistMenu = ({
    steamId,
    onClose
}: SteamIDProps & SubMenuProps) => {
    const { t } = useTranslation();

    const mutation = useMutation({
        ...addWhitelistMutation(),
        onSuccess: () => {
            onClose();
        },
        onError: (err: Error) => {
            logError(err);
            onClose();
        }
    });

    return (
        <IconMenuItem
            leftIcon={<NotificationsPausedOutlinedIcon color={'primary'} />}
            label={t('player_table.menu.whitelist_label')}
            onClick={async () => {
                mutation.mutate({ steamId });
            }}
        />
    );
};
