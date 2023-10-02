import React, { useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { IconMenuItem, NestedMenuItem } from 'mui-nested-menu';
import ArrowRightOutlinedIcon from '@mui/icons-material/ArrowRightOutlined';
import HowToRegIcon from '@mui/icons-material/HowToReg';
import { callVote, kickReasons } from '../../api';
import { SteamIDProps, SubMenuProps } from './common';
import { logError } from '../../util';

export const CallVoteMenu = ({
    contextMenuPos,
    steam_id,
    onClose
}: SubMenuProps & SteamIDProps) => {
    const { t } = useTranslation();

    const onCallVote = useCallback(
        async (reason: kickReasons) => {
            try {
                await callVote(steam_id, reason);
            } catch (e) {
                logError(e);
            } finally {
                onClose();
            }
        },
        [onClose, steam_id]
    );

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
