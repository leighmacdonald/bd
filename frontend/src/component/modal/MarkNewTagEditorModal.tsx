import { useCallback, useMemo, useState } from 'react';
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
import { logError } from '../../util';
import { CancelButton } from '../CancelButton.tsx';
import SaveButton from '../SaveButton.tsx';
import { saveUserSettings, UserSettings } from '../../api.ts';
import { useMutation } from '@tanstack/react-query';

interface MarkNewTagEditorProps {
    steam_id: string;
    onMarkAs: (steamId: string, attrs: string[]) => Promise<void>;
    settings: UserSettings;
}

export const MarkNewTagEditorModal = NiceModal.create<MarkNewTagEditorProps>(
    ({ steam_id, onMarkAs, settings }) => {
        const [tag, setTag] = useState<string>('');
        const { t } = useTranslation();
        const modal = useModal();

        const saveSettings = useMutation({
            mutationFn: saveUserSettings
        });

        const onSaveMarkWithNewTag = useCallback(async () => {
            try {
                await onMarkAs(steam_id, [tag]);

                saveSettings.mutate({
                    ...settings,
                    unique_tags: [tag, ...settings.unique_tags]
                });
            } catch (e) {
                logError(`Error updating note: ${e}`);
            } finally {
                await modal.hide();
            }
        }, [onMarkAs, steam_id, tag, settings, modal]);

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
                    <CancelButton onClick={modal.hide} />
                    <SaveButton
                        onClick={onSaveMarkWithNewTag}
                        disabled={!validTag}
                    />
                </DialogActions>
            </Dialog>
        );
    }
);

export default MarkNewTagEditorModal;
