import { useTranslation } from 'react-i18next';
import React, { useCallback } from 'react';
import { markUser } from '../../api';
import { IconMenuItem, NestedMenuItem } from 'mui-nested-menu';
import ArrowRightOutlinedIcon from '@mui/icons-material/ArrowRightOutlined';
import FlagIcon from '@mui/icons-material/Flag';
import { SteamIDProps, SubMenuProps } from './common';
import { logError } from '../../util';

interface MarkMenuProps {
    unique_tags: string[];
}

export const MarkMenu = ({
    contextMenuPos,
    unique_tags,
    steam_id,
    onClose
}: MarkMenuProps & SteamIDProps & SubMenuProps) => {
    const { t } = useTranslation();

    const onMarkAs = useCallback(
        async (steamId: string, attrs: string[]) => {
            try {
                await markUser(steamId, attrs);
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
            leftIcon={<FlagIcon color={'primary'} />}
            label={t('player_table.menu.mark_label')}
            parentMenuOpen={contextMenuPos !== null}
        >
            {[
                ...unique_tags.filter((tag) => tag.toLowerCase() != 'new'),
                'new'
            ].map((attr) => {
                return (
                    <IconMenuItem
                        leftIcon={<FlagIcon color={'primary'} />}
                        onClick={async () => {
                            await onMarkAs(steam_id, [attr]);
                        }}
                        label={attr}
                        key={`tag-${steam_id}-${attr}`}
                    />
                );
            })}
        </NestedMenuItem>
    );
};
