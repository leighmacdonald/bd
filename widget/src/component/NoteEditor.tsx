import React, { useCallback } from 'react';
import Dialog from '@mui/material/Dialog';
import {
    Button,
    DialogActions,
    DialogContent,
    DialogTitle,
    TextField
} from '@mui/material';
import Stack from '@mui/material/Stack';

interface NoteEditorProps {
    open: boolean;
    setOpen: (open: boolean) => void;
    notes: string;
    setNotes: (note: string) => void;
    steamId: bigint;
    setSteamId: (steamId: bigint) => void;
    onSave: (steamId: bigint, note: string) => void;
}

export const NoteEditor = ({
    open,
    setOpen,
    notes,
    setNotes,
    steamId,
    onSave
}: NoteEditorProps) => {
    const handleClose = () => setOpen(false);

    const handleSave = useCallback(async () => {
        await onSave(steamId, notes);
        handleClose();
    }, [onSave, steamId, notes]);

    return (
        <Dialog open={open} onClose={handleClose} fullWidth>
            <DialogTitle>Edit Player Notes</DialogTitle>
            <DialogContent>
                <Stack spacing={1} padding={1}>
                    <TextField
                        id="outlined-multiline-flexible"
                        label="Note"
                        fullWidth
                        minRows={10}
                        value={notes}
                        onChange={(evt) => {
                            setNotes(evt.target.value);
                        }}
                        multiline
                    />
                </Stack>
            </DialogContent>
            <DialogActions>
                <Button onClick={handleClose}>Cancel</Button>
                <Button onClick={handleSave}>Save</Button>
            </DialogActions>
        </Dialog>
    );
};
