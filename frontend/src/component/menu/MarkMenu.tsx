import { useCallback, useContext } from 'react';
import { useTranslation } from 'react-i18next';
import { markUser } from '../../api';
import { IconMenuItem, NestedMenuItem } from 'mui-nested-menu';
import ArrowRightOutlinedIcon from '@mui/icons-material/ArrowRightOutlined';
import FlagIcon from '@mui/icons-material/Flag';
import { SteamIDProps, SubMenuProps } from './common';
import { logError } from '../../util';
import NiceModal, { useModal } from '@ebay/nice-modal-react';
import { SettingsContext } from '../../context/SettingsContext';
import { ModalMarkNewTag } from '../modal';

export const MarkMenu = ({
    contextMenuPos,
    steam_id,
    onClose
}: SteamIDProps & SubMenuProps) => {
    const { t } = useTranslation();
    const modal = useModal(ModalMarkNewTag);
    const { settings } = useContext(SettingsContext);

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

export default MarkMenu;
