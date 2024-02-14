import { List, UserSettings } from '../api';
import {
    ChangeEvent,
    Dispatch,
    SetStateAction,
    useCallback,
    useState
} from 'react';
import Dialog from '@mui/material/Dialog';
import {
    Checkbox,
    DialogActions,
    DialogContent,
    DialogTitle,
    FormControlLabel,
    FormGroup,
    TextField
} from '@mui/material';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { useTranslation } from 'react-i18next';
import NiceModal, { muiDialog, useModal } from '@ebay/nice-modal-react';
import cloneDeep from 'lodash/cloneDeep';
import { CancelButton, ResetButton, SaveButton } from './Buttons';

interface SettingsListProps {
    list: List;
    rowIndex: number;
    setNewSettings: Dispatch<SetStateAction<UserSettings>>;
}

export const SettingsListEditor = NiceModal.create<SettingsListProps>(
    ({ list, rowIndex, setNewSettings }) => {
        const [newList, setNewList] = useState<List>({ ...list });
        const modal = useModal();
        const { t } = useTranslation();

        const handleClose = useCallback(async () => {
            await modal.hide();
        }, [modal]);

        const handleSave = useCallback(async () => {
            setNewSettings((prevState) => {
                const s = prevState;
                s.lists[rowIndex] = newList;
                return s;
            });
            await modal.hide();
        }, [modal, newList, rowIndex, setNewSettings]);

        const onEnabledChanged = (
            _: ChangeEvent<HTMLInputElement>,
            enabled: boolean
        ) => {
            setNewList({ ...newList, enabled });
        };

        const handleReset = useCallback(() => {
            setNewList(cloneDeep(list));
        }, [list]);

        const onNameChanged = (event: ChangeEvent<HTMLInputElement>) => {
            setNewList({ ...newList, name: event.target.value });
        };

        const onUrlChanged = (event: ChangeEvent<HTMLInputElement>) => {
            setNewList({ ...newList, url: event.target.value });
        };

        return (
            <Dialog fullWidth {...muiDialog(modal)}>
                <DialogTitle component={Typography} variant={'h1'}>
                    {list.url == ''
                        ? t('settings.list_editor.create_title')
                        : `${t('settings.list_editor.edit_title')} ${
                              newList.name
                          }`}
                </DialogTitle>
                <DialogContent dividers>
                    <Stack>
                        <FormGroup>
                            <FormControlLabel
                                control={
                                    <Checkbox
                                        checked={newList.enabled}
                                        onChange={onEnabledChanged}
                                    />
                                }
                                label={t('settings.list_editor.enabled_label')}
                            />
                        </FormGroup>

                        <TextField
                            value={newList.name}
                            onChange={onNameChanged}
                        />

                        <TextField
                            fullWidth
                            label={'Update URL'}
                            value={newList.url}
                            onChange={onUrlChanged}
                        />
                    </Stack>
                </DialogContent>

                <DialogActions>
                    <CancelButton onClick={handleClose} />
                    <ResetButton onClick={handleReset} />
                    <SaveButton onClick={handleSave} />
                </DialogActions>
            </Dialog>
        );
    }
);
