import { Link, steamIdFormat, useUserSettings } from '../api';
import React, { ChangeEvent, useCallback, useEffect, useState } from 'react';
import Dialog from '@mui/material/Dialog';
import {
    Button,
    Checkbox,
    DialogActions,
    DialogContent,
    DialogTitle,
    FormControl,
    FormControlLabel,
    FormGroup,
    InputLabel,
    Select,
    SelectChangeEvent,
    TextField
} from '@mui/material';
import Stack from '@mui/material/Stack';
import CloseIcon from '@mui/icons-material/Close';
import CheckIcon from '@mui/icons-material/Check';
import { inputValidator } from './SettingsEditor';
import MenuItem from '@mui/material/MenuItem';
import Typography from '@mui/material/Typography';
import { Trans, useTranslation } from 'react-i18next';
import NiceModal, { muiDialog, useModal } from '@ebay/nice-modal-react';
import cloneDeep from 'lodash/cloneDeep';
import RestartAltIcon from '@mui/icons-material/RestartAlt';

interface SettingsLinkProps {
    link: Link;
    rowIndex: number;
    validator?: inputValidator;
}

export const SettingsLinkEditor = NiceModal.create<SettingsLinkProps>(
    ({ link, rowIndex }) => {
        const modal = useModal();
        const { t } = useTranslation();
        const { setNewSettings } = useUserSettings();

        const [newLink, setNewLink] = useState<Link>(cloneDeep(link));

        const handleReset = useCallback(() => {
            setNewLink(cloneDeep(link));
        }, [link]);

        useEffect(() => {
            handleReset();
        }, [handleReset, link]);

        const onEnabledChanged = (
            _: ChangeEvent<HTMLInputElement>,
            enabled: boolean
        ) => {
            setNewLink({ ...newLink, enabled });
        };

        const onNameChanged = useCallback(
            (event: ChangeEvent<HTMLInputElement>) => {
                setNewLink({ ...newLink, name: event.target.value });
            },
            [newLink]
        );

        const handleSave = useCallback(async () => {
            setNewSettings((prevState) => {
                prevState.links[rowIndex] = newLink;
                return prevState;
            });
            await modal.hide();
        }, [modal, newLink, rowIndex, setNewSettings]);

        const onUrlChanged = (event: ChangeEvent<HTMLInputElement>) => {
            setNewLink({ ...newLink, url: event.target.value });
        };

        const onFormatChanged = (event: SelectChangeEvent) => {
            setNewLink({
                ...newLink,
                id_format: event.target.value as steamIdFormat
            });
        };

        return (
            <Dialog fullWidth {...muiDialog(modal)}>
                <DialogTitle component={Typography} variant={'h1'}>
                    {link.url == ''
                        ? t('settings.link_editor.create_title')
                        : `${t('settings.link_editor.edit_title')} ${
                              link.name
                          }`}
                </DialogTitle>
                <DialogContent dividers>
                    <Stack spacing={2}>
                        <FormGroup>
                            <FormControlLabel
                                control={
                                    <Checkbox
                                        checked={newLink.enabled}
                                        onChange={onEnabledChanged}
                                    />
                                }
                                label={t('settings.link_editor.enabled_label')}
                            />
                        </FormGroup>

                        <TextField
                            value={newLink.name}
                            onChange={onNameChanged}
                        />
                        <FormControl fullWidth>
                            <InputLabel id="steam_id_format-select-label">
                                <Trans
                                    i18nKey={
                                        'settings.link_editor.steam_id_format'
                                    }
                                />
                            </InputLabel>
                            <Select<steamIdFormat>
                                labelId="steam_id_format-select-label"
                                id="steam_id_format-select"
                                value={newLink.id_format}
                                onChange={onFormatChanged}
                            >
                                {(
                                    [
                                        'steam64',
                                        'steam3',
                                        'steam32',
                                        'steam'
                                    ] as steamIdFormat[]
                                ).map((s) => (
                                    <MenuItem value={s} key={`steam-fmt-${s}`}>
                                        {s}
                                    </MenuItem>
                                ))}
                            </Select>
                        </FormControl>
                        <TextField
                            fullWidth
                            value={newLink.url}
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
                        <Trans i18nKey={'button.ok'} />
                    </Button>
                </DialogActions>
            </Dialog>
        );
    }
);
