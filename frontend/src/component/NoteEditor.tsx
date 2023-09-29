import React, { useCallback, useState } from 'react';
import Dialog from '@mui/material/Dialog';
import {
    Button,
    DialogActions,
    DialogContent,
    DialogTitle,
    TextField
} from '@mui/material';
import Stack from '@mui/material/Stack';
import CloseIcon from '@mui/icons-material/Close';
import SaveIcon from '@mui/icons-material/Save';
import { Trans, useTranslation } from 'react-i18next';
import ClearIcon from '@mui/icons-material/Clear';
import { saveUserNote } from '../api';
import NiceModal, { muiDialog, useModal } from '@ebay/nice-modal-react';

interface NoteEditorProps {
    notes: string;
    steamId: string;
}

export const NoteEditor = NiceModal.create<NoteEditorProps>(
    ({ steamId, notes }) => {
        const [newNotes, setNewNotes] = useState<string>(notes);
        const { t } = useTranslation();
        const modal = useModal();

        const onSaveNotes = useCallback(async () => {
            try {
                await saveUserNote(steamId, newNotes);
                await modal.hide();
            } catch (e) {
                console.log(`Error updating note: ${e}`);
            }
        }, [newNotes]);

        return (
            <Dialog fullWidth {...muiDialog(modal)}>
                <DialogTitle>
                    <Trans i18nKey={'player_table.notes.title'} />
                </DialogTitle>
                <DialogContent>
                    <Stack spacing={1} padding={0}>
                        <TextField
                            id="notes-editor-field"
                            label={t('player_table.notes.note_label')}
                            fullWidth
                            minRows={10}
                            value={newNotes}
                            onChange={(evt) => {
                                setNewNotes(evt.target.value);
                            }}
                            multiline
                        />
                    </Stack>
                </DialogContent>
                <DialogActions>
                    <Button
                        startIcon={<ClearIcon />}
                        color={'warning'}
                        variant={'contained'}
                        onClick={() => {
                            setNewNotes('');
                        }}
                    >
                        <Trans i18nKey={'button.clear'} />
                    </Button>
                    <Button
                        startIcon={<CloseIcon />}
                        color={'error'}
                        variant={'contained'}
                        onClick={modal.hide}
                    >
                        <Trans i18nKey={'button.cancel'} />
                    </Button>
                    <Button
                        startIcon={<SaveIcon />}
                        color={'success'}
                        variant={'contained'}
                        onClick={onSaveNotes}
                    >
                        <Trans i18nKey={'button.save'} />
                    </Button>
                </DialogActions>
            </Dialog>
        );
    }
);
