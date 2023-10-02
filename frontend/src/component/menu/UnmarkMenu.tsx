import React, { useCallback } from 'react';
import { unmarkUser } from '../../api';
import { IconMenuItem } from 'mui-nested-menu';
import DeleteOutlinedIcon from '@mui/icons-material/DeleteOutlined';
import { SteamIDProps, SubMenuProps } from './common';
import { logError } from '../../util';

export const UnmarkMenu = ({
    steam_id,
    onClose
}: SteamIDProps & SubMenuProps) => {
    const onUnmark = useCallback(
        async (steamId: string) => {
            try {
                await unmarkUser(steamId);
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
            leftIcon={<DeleteOutlinedIcon color={'primary'} />}
            label={'Unmark'}
            onClick={async () => {
                await onUnmark(steam_id);
            }}
        />
    );
};
