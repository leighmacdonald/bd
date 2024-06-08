import { IconMenuItem } from 'mui-nested-menu';
import DeleteOutlinedIcon from '@mui/icons-material/DeleteOutlined';
import { SteamIDProps, SubMenuProps } from './common';
import { logError } from '../../util';
import { useMutation } from '@tanstack/react-query';
import { unmarkUserMutation } from '../../api.ts';

export const UnmarkMenu = ({
    steamId,
    onClose
}: SteamIDProps & SubMenuProps) => {
    const mutation = useMutation({
        ...unmarkUserMutation(steamId),

        onSuccess: () => {
            console.log('Unmaked user');
            onClose();
        },
        onError: (err: Error) => {
            logError(err);
            onClose();
        }
    });

    return (
        <IconMenuItem
            leftIcon={<DeleteOutlinedIcon color={'primary'} />}
            label={'Unmark'}
            onClick={async () => {
                mutation.mutate();
            }}
        />
    );
};

export default UnmarkMenu;
