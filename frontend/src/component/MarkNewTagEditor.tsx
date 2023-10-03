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

interface MarkNewTagEditorProps {
    steam_id: string;
    onMarkAs: (steamId: string, attrs: string[]) => Promise<void>;
}

export const MarkNewTagEditor = NiceModal.create<MarkNewTagEditorProps>(
    ({ steam_id, onMarkAs }) => {
        const [tag, setTag] = useState<string>('');
        const { t } = useTranslation();
        const modal = useModal();

        const onSaveMarkWithNewTag = useCallback(async () => {
            try {
                await onMarkAs(steam_id, [tag]);
            } catch (e) {
                logError(`Error updating note: ${e}`);
            } finally {
                await modal.hide();
            }
        }, [onMarkAs, steam_id, tag, modal]);

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
                    <SaveButton
                        onClick={onSaveMarkWithNewTag}
                        disabled={!validTag}
                    />
                </DialogActions>
            </Dialog>
        );
    }
);
