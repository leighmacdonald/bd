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

interface NoteEditorProps {
    open: boolean;
    setOpen: (open: boolean) => void;
    notes: string;
    setNotes: (note: string) => void;
    steamId: string;
    setSteamId: (steamId: string) => void;
    onSave: (steamId: string, note: string) => void;
}

export const NoteEditor = ({
    open,
    setOpen,
    notes,
    setNotes,
    steamId,
    onSave
}: NoteEditorProps) => {
    const [newNotes, setNewNotes] = useState<string>(notes);
    const handleClose = useCallback(() => setOpen(false), [setOpen]);
    const { t } = useTranslation();

    const handleSave = useCallback(async () => {
        await onSave(steamId, notes);
        setNotes(newNotes);
        handleClose();
    }, [onSave, steamId, notes, setNotes, newNotes, handleClose]);

    return (
        <Dialog open={open} onClose={handleClose} fullWidth>
            <DialogTitle>
                <Trans i18nKey={'notes.title'} />
            </DialogTitle>
            <DialogContent>
                <Stack spacing={1} padding={0}>
                    <TextField
                        id="notes-editor-field"
                        label={t('notes.note_label')}
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
                    startIcon={<CloseIcon />}
                    color={'error'}
                    variant={'contained'}
                    onClick={handleClose}
                >
                    <Trans i18nKey={'button.cancel'} />
                </Button>
                <Button
                    startIcon={<SaveIcon />}
                    color={'success'}
                    variant={'contained'}
                    onClick={handleSave}
                >
                    <Trans i18nKey={'button.save'} />
                </Button>
            </DialogActions>
        </Dialog>
    );
};
