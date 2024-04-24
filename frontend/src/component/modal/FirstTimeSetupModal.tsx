import { useCallback, useState } from 'react';
import Dialog from '@mui/material/Dialog';
import {
    DialogActions,
    DialogContent,
    DialogTitle,
    TextField
} from '@mui/material';
import Stack from '@mui/material/Stack';
import { Trans, useTranslation } from 'react-i18next';
import { FirstTimeSetup, saveFirstTimeSetup } from '../../api.ts';
import NiceModal, { muiDialog, useModal } from '@ebay/nice-modal-react';
import { logError } from '../../util.ts';
import CancelButton from '../CancelButton.tsx';
import SaveButton from '../SaveButton.tsx';
import ClearButton from '../ClearButton.tsx';

interface NoteEditorProps {
    default_tf2_dir: string;
}

export const FirstTimeSetupModal = NiceModal.create<NoteEditorProps>(
    ({ default_tf2_dir }) => {
        const [setupValues, setSetupValues] = useState<FirstTimeSetup>({
            tf2_dir: default_tf2_dir,
            steam_id: ''
        });
        const { t } = useTranslation();
        const modal = useModal();

        const onSaveNotes = useCallback(async () => {
            try {
                await saveFirstTimeSetup(setupValues);
                await modal.hide();
            } catch (e) {
                logError(`Error updating note: ${e}`);
            }
        }, [setupValues, modal]);

        return (
            <Dialog fullWidth {...muiDialog(modal)}>
                <DialogTitle>
                    <Trans i18nKey={'player_table.notes.title'} />
                </DialogTitle>
                <DialogContent>
                    <Stack spacing={1} padding={0}>
                        <TextField
                            id="notes-editor-field"
                            label={t('setup.steam_id_label')}
                            fullWidth
                            minRows={10}
                            value={setupValues.steam_id}
                            onChange={(evt) => {
                                setSetupValues((prevState) => {
                                    return {
                                        ...prevState,
                                        steam_id: evt.target.value
                                    };
                                });
                            }}
                            multiline
                        />
                    </Stack>
                </DialogContent>
                <DialogActions>
                    <ClearButton
                        onClick={() => {
                            setSetupValues({
                                steam_id: '',
                                tf2_dir: default_tf2_dir
                            });
                        }}
                    />
                    <CancelButton onClick={modal.hide} />
                    <SaveButton onClick={onSaveNotes} />
                </DialogActions>
            </Dialog>
        );
    }
);

export default FirstTimeSetupModal;
