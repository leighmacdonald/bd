import { List, UserSettings } from '../api';
import React, {
    ChangeEvent,
    Dispatch,
    SetStateAction,
    useCallback,
    useState
} from 'react';
import Dialog from '@mui/material/Dialog';
import {
    Button,
    Checkbox,
    DialogActions,
    DialogContent,
    DialogTitle,
    TextField
} from '@mui/material';
import Stack from '@mui/material/Stack';
import CloseIcon from '@mui/icons-material/Close';
import CheckIcon from '@mui/icons-material/Check';
import Typography from '@mui/material/Typography';
import { Trans, useTranslation } from 'react-i18next';
import NiceModal, { muiDialog, useModal } from '@ebay/nice-modal-react';
import RestartAltIcon from '@mui/icons-material/RestartAlt';
import cloneDeep from 'lodash/cloneDeep';

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
            <Dialog {...muiDialog(modal)}>
                <DialogTitle component={Typography} variant={'h1'}>
                    {list.url == ''
                        ? t('settings.list_editor.create_title')
                        : `${t('settings.list_editor.edit_title')} ${
                              newList.name
                          }`}
                </DialogTitle>
                <DialogContent dividers>
                    <Stack>
                        <Checkbox
                            checked={newList.enabled}
                            onChange={onEnabledChanged}
                        />
                        <TextField
                            value={newList.name}
                            onChange={onNameChanged}
                        />
                        <TextField
                            fullWidth
                            value={newList.url}
                            onChange={onUrlChanged}
                        />
                    </Stack>
                </DialogContent>

                <DialogActions>
                    <Button
                        onClick={modal.hide}
                        startIcon={<CloseIcon />}
                        color={'error'}
                        variant={'contained'}
                    >
                        <Trans i18nKey={'button.cancel'} />
                    </Button>
                    <Button
                        onClick={handleReset}
                        startIcon={<RestartAltIcon />}
                        color={'warning'}
                        variant={'contained'}
                    >
                        <Trans i18nKey={'button.reset'} />
                    </Button>
                    <Button
                        onClick={handleSave}
                        startIcon={<CheckIcon />}
                        color={'success'}
                        variant={'contained'}
                    >
                        <Trans i18nKey={'button.save'} />
                    </Button>
                </DialogActions>
            </Dialog>
        );
    }
);
