import { useCallback, useContext } from 'react';
import { useTranslation } from 'react-i18next';
import { markUserMutation } from '../../api';
import { IconMenuItem, NestedMenuItem } from 'mui-nested-menu';
import ArrowRightOutlinedIcon from '@mui/icons-material/ArrowRightOutlined';
import FlagIcon from '@mui/icons-material/Flag';
import { SteamIDProps, SubMenuProps } from './common';
import { logError } from '../../util';
import NiceModal, { useModal } from '@ebay/nice-modal-react';
import { SettingsContext } from '../../context/SettingsContext';
import { ModalMarkNewTag } from '../modal';
import { useMutation } from '@tanstack/react-query';

export const MarkMenu = ({
    contextMenuPos,
    steamId,
    onClose
}: SteamIDProps & SubMenuProps) => {
    const { t } = useTranslation();
    const modal = useModal(ModalMarkNewTag);
    const { settings } = useContext(SettingsContext);

    const mutation = useMutation({
        ...markUserMutation(steamId),
        onSuccess: () => {
            onClose();
            console.log('Marked user');
        },
        onError: (error) => {
            logError(error);
            onClose();
        }
    });

    const onMarkAsNew = useCallback(async () => {
        try {
            await NiceModal.show(ModalMarkNewTag, {
                steamId: steamId,
                mutation
            });
        } catch (e) {
            logError(e);
        } finally {
            await modal.hide();
        }
    }, [modal, mutation, steamId]);

    return (
        <NestedMenuItem
            rightIcon={<ArrowRightOutlinedIcon />}
            leftIcon={<FlagIcon color={'primary'} />}
            label={t('player_table.menu.mark_label')}
            parentMenuOpen={contextMenuPos !== null}
        >
            {[
                ...settings.unique_tags.filter(
                    (tag: string) => tag.toLowerCase() != 'new...'
                ),
                'new...'
            ].map((attr) => {
                return (
                    <IconMenuItem
                        onClick={async () => {
                            if (attr == 'new...') {
                                await onMarkAsNew();
                            } else {
                                mutation.mutate({ attrs: [attr] });
                            }
                        }}
                        label={attr}
                        key={`tag-${steamId}-${attr}`}
                    />
                );
            })}
        </NestedMenuItem>
    );
};
