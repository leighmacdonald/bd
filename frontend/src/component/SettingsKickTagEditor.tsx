import React, { useCallback, useMemo, useState } from 'react';
import Dialog from '@mui/material/Dialog';
import {
    DialogActions,
    DialogContent,
    DialogTitle,
    TextField
} from '@mui/material';
import Stack from '@mui/material/Stack';
import { Trans, useTranslation } from 'react-i18next';
import NiceModal, { muiDialog, useModal } from '@ebay/nice-modal-react';
import { logError } from '../util';
import { CancelButton, SaveButton } from './Buttons';
import { UserSettings } from '../api';
import { sortedUniq } from 'lodash';

interface KickTagEditorProps {
    setNewSettings: React.Dispatch<React.SetStateAction<UserSettings>>;
}

export const SettingsKickTagEditor = NiceModal.create<KickTagEditorProps>(
    ({ setNewSettings }) => {
        const [tag, setTag] = useState<string>('');
        const { t } = useTranslation();
        const modal = useModal();

        const onSaveTag = useCallback(async () => {
            try {
                setNewSettings((prevState) => {
                    return {
                        ...prevState,
                        kick_tags: sortedUniq([...prevState.kick_tags, tag])
                    };
                });
                setTag('');
            } catch (e) {
                logError(`Error updating note: ${e}`);
            } finally {
                await modal.hide();
            }
        }, [setNewSettings, tag, modal]);

        const validTag = useMemo(() => {
            return tag.length > 0 && !tag.match(/\s/);
        }, [tag]);

        return (
            <Dialog fullWidth {...muiDialog(modal)}>
                <DialogTitle>
                    <Trans i18nKey={'mark_new_tag.title'} />
                </DialogTitle>
                <DialogContent>
                    <Stack spacing={1} padding={0}>
                        <TextField
                            error={tag.length > 0 && !validTag}
                            id="new-tag-editor-field"
                            label={t('mark_new_tag.tag')}
                            fullWidth
                            value={tag}
                            onChange={(evt) => {
                                setTag(evt.target.value);
                            }}
                        />
                    </Stack>
                </DialogContent>
                <DialogActions>
                    <CancelButton />
                    <SaveButton onClick={onSaveTag} disabled={!validTag} />
                </DialogActions>
            </Dialog>
        );
    }
);