import { useMemo, useState } from 'react';
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
import { CancelButton } from '../CancelButton.tsx';
import SaveButton from '../SaveButton.tsx';

export const MarkNewTagEditorModal = NiceModal.create(() => {
    const [tag, setTag] = useState<string>('');
    const { t } = useTranslation();
    const modal = useModal();

    const onSaveMarkWithNewTag = () => {
        modal.resolve({ tag });
    };

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
});
