import { useTranslation } from 'react-i18next';
import React, { useCallback } from 'react';
import { markUser } from '../../api';
import { IconMenuItem, NestedMenuItem } from 'mui-nested-menu';
import ArrowRightOutlinedIcon from '@mui/icons-material/ArrowRightOutlined';
import FlagIcon from '@mui/icons-material/Flag';
import { SteamIDProps, SubMenuProps } from './common';
import { logError } from '../../util';
import NiceModal, { useModal } from '@ebay/nice-modal-react';
import { ModalMarkNewTag } from '../../modals';

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
    const modal = useModal(ModalMarkNewTag);

    const onMarkAs = useCallback(
        async (sid: string, attrs: string[]) => {
            try {
                await markUser(sid, attrs);
            } catch (e) {
                logError(e);
            } finally {
                onClose();
            }
        },
        [onClose]
    );

    const onMarkAsNew = useCallback(async () => {
        try {
            await NiceModal.show(ModalMarkNewTag, { steam_id, onMarkAs });
        } catch (e) {
            logError(e);
        } finally {
            await modal.hide();
        }
    }, [modal, onMarkAs, steam_id]);

    return (
        <NestedMenuItem
            rightIcon={<ArrowRightOutlinedIcon />}
            leftIcon={<FlagIcon color={'primary'} />}
            label={t('player_table.menu.mark_label')}
            parentMenuOpen={contextMenuPos !== null}
        >
            {[
                ...unique_tags.filter((tag) => tag.toLowerCase() != 'new...'),
                'new...'
            ].map((attr) => {
                return (
                    <IconMenuItem
                        leftIcon={<FlagIcon color={'primary'} />}
                        onClick={async () => {
                            if (attr == 'new...') {
                                await onMarkAsNew();
                            } else {
                                await onMarkAs(steam_id, [attr]);
                            }
                        }}
                        label={attr}
                        key={`tag-${steam_id}-${attr}`}
                    />
                );
            })}
        </NestedMenuItem>
    );
};
