import { useTranslation } from 'react-i18next';
import { IconMenuItem, NestedMenuItem } from 'mui-nested-menu';
import ArrowRightOutlinedIcon from '@mui/icons-material/ArrowRightOutlined';
import HowToRegIcon from '@mui/icons-material/HowToReg';
import { callVoteMutation, kickReasons } from '../../api';
import { SteamIDProps, SubMenuProps } from './common';
import { logError } from '../../util';
import { useMutation } from '@tanstack/react-query';

export const CallVoteMenu = ({
    contextMenuPos,
    steamId,
    onClose
}: SubMenuProps & SteamIDProps) => {
    const { t } = useTranslation();

    const mutation = useMutation({
        ...callVoteMutation(steamId),
        onSuccess: () => {
            onClose();
            console.log('Called vote');
        },
        onError: (error: Error) => {
            logError(error);
        }
    });

    const onCallVote = async (reason: kickReasons) => {
        mutation.mutate({ reason });
    };

    return (
        <NestedMenuItem
            rightIcon={<ArrowRightOutlinedIcon />}
            leftIcon={<HowToRegIcon color={'primary'} />}
            label={t('player_table.menu.vote_label')}
            parentMenuOpen={contextMenuPos !== null}
        >
            {(['idle', 'scamming', 'cheating', 'other'] as kickReasons[]).map(
                (reason: kickReasons) => (
                    <IconMenuItem
                        key={`vote-type-icon-${reason}`}
                        onClick={async () => {
                            await onCallVote(reason);
                        }}
                        label={reason}
                    />
                )
            )}
        </NestedMenuItem>
    );
};
