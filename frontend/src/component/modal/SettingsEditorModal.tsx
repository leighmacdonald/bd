import { SyntheticEvent, useCallback, useEffect, useState } from 'react';
import {
    Accordion,
    AccordionDetails,
    AccordionSummary,
    DialogActions,
    DialogContent,
    DialogTitle,
    useTheme
} from '@mui/material';
import Dialog from '@mui/material/Dialog';
import { Link, List, saveUserSettings, UserSettings } from '../../api.ts';
import Grid from '@mui/material/Unstable_Grid2';
import Stack from '@mui/material/Stack';
import IconButton from '@mui/material/IconButton';
import AlarmOnIcon from '@mui/icons-material/AlarmOn';
import AlarmOffIcon from '@mui/icons-material/AlarmOff';
import EditIcon from '@mui/icons-material/Edit';
import DeleteIcon from '@mui/icons-material/Delete';
import Typography from '@mui/material/Typography';
import Box from '@mui/material/Box';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import { Trans, useTranslation } from 'react-i18next';
import {
    isValidUrl,
    logError,
    makeValidatorLength,
    validatorAddress,
    validatorSteamID
} from '../../util.ts';
import NiceModal, { muiDialog, useModal } from '@ebay/nice-modal-react';
import { ModalSettings, ModalSettingsList } from './index.ts';
import SettingsCheckBox from '../SettingsCheckbox.tsx';
import SettingsMultiSelect from '../SettingsMultiSelect.tsx';
import SettingsTextBox from '../SettingsTextBox.tsx';
import CancelButton from '../CancelButton.tsx';
import SaveButton from '../SaveButton.tsx';
import ResetButton from '../ResetButton.tsx';

const SettingsEditorModal = NiceModal.create(
    ({ settings }: { settings: UserSettings }) => {
        const settingsModal = useModal(ModalSettings);
        const { t } = useTranslation();
        const theme = useTheme();
        const modal = useModal();

        // structuredClone not supported on steam CEF version 85...
        const [newSettings, setNewSettings] = useState<UserSettings>({
            ...settings
        });

        const handleReset = useCallback(() => {
            setNewSettings({ ...settings });
        }, [settings, setNewSettings]);

        const onOpenLink = useCallback(
            async (link: Link, rowIndex: number) => {
                try {
                    await settingsModal.show({
                        link,
                        rowIndex,
                        setNewSettings
                    });
                } catch (e) {
                    logError(e);
                } finally {
                    console.log(newSettings.links);
                    await settingsModal.hide();
                }
            },
            [settingsModal, newSettings.links]
        );

        const onOpenList = useCallback(
            async (list: List, rowIndex: number) => {
                try {
                    await NiceModal.show(ModalSettingsList, {
                        list,
                        rowIndex,
                        setNewSettings
                    });
                } catch (e) {
                    logError(e);
                } finally {
                    console.log(newSettings.lists);
                    await settingsModal.hide();
                }
            },
            [settingsModal, newSettings.lists]
        );

        useEffect(() => {
            handleReset();
        }, [handleReset]);

        const onSubmit = useCallback(async () => {
            try {
                await saveUserSettings(newSettings);
                modal.resolve(newSettings);
                await modal.hide();
            } catch (reason) {
                modal.reject(reason);
                logError(reason);
            }
        }, [newSettings, settingsModal]);

        const toggleList = useCallback(
            (i: number) => {
                setNewSettings((us: UserSettings) => {
                    const s = { ...us };
                    s.lists[i].enabled = !s.lists[i].enabled;
                    return s;
                });
            },
            [setNewSettings]
        );

        const toggleLink = useCallback(
            (i: number) => {
                setNewSettings((us: UserSettings) => {
                    const s = { ...us };
                    s.links[i].enabled = !s.links[i].enabled;
                    return s;
                });
            },
            [setNewSettings]
        );

        const deleteLink = useCallback(
            (i: number) => {
                const newLinks = newSettings.links.filter(
                    (_: Link, idx: number) => idx != i
                );
                setNewSettings({ ...newSettings, links: newLinks });
            },
            [newSettings, setNewSettings]
        );

        const deleteList = useCallback(
            (i: number) => {
                const newList = newSettings.lists.filter(
                    (_: List, idx: number) => idx != i
                );
                setNewSettings({ ...newSettings, lists: newList });
            },
            [newSettings, setNewSettings]
        );

        const [expanded, setExpanded] = useState<string | false>('general');

        const handleChange =
            (panel: string) => (_: SyntheticEvent, newExpanded: boolean) => {
                setExpanded(newExpanded ? panel : false);
            };

        return (
            <Dialog fullWidth {...muiDialog(settingsModal)}>
                <DialogTitle component={Typography} variant={'h1'}>
                    {t('settings.label')}
                </DialogTitle>
                <DialogContent dividers={true} sx={{ padding: 0 }}>
                    <Accordion
                        TransitionProps={{ unmountOnExit: true }}
                        expanded={expanded === 'general'}
                        onChange={handleChange('general')}
                    >
                        <AccordionSummary
                            style={{
                                backgroundColor: theme.palette.background.paper
                            }}
                            expandIcon={<ExpandMoreIcon />}
                            aria-controls="general-content"
                            id="general-header"
                        >
                            <Typography sx={{ width: '33%', flexShrink: 0 }}>
                                {t('settings.general.label')}
                            </Typography>
                            <Typography sx={{ color: 'text.secondary' }}>
                                {t('settings.general.description')}
                            </Typography>
                        </AccordionSummary>
                        <AccordionDetails>
                            <Grid container spacing={1}>
                                <Grid xs={6}>
                                    <SettingsCheckBox
                                        label={t(
                                            'settings.general.chat_warnings_label'
                                        )}
                                        tooltip={t(
                                            'settings.general.chat_warnings_tooltip'
                                        )}
                                        enabled={
                                            newSettings.chat_warnings_enabled
                                        }
                                        setEnabled={(chat_warnings_enabled) => {
                                            setNewSettings({
                                                ...newSettings,
                                                chat_warnings_enabled
                                            });
                                        }}
                                    />
                                </Grid>
                                <Grid xs={6}>
                                    <SettingsCheckBox
                                        label={t(
                                            'settings.general.kicker_enabled_label'
                                        )}
                                        tooltip={t(
                                            'settings.general.kicker_enabled_tooltip'
                                        )}
                                        enabled={newSettings.kicker_enabled}
                                        setEnabled={(kicker_enabled) => {
                                            setNewSettings({
                                                ...newSettings,
                                                kicker_enabled
                                            });
                                        }}
                                    />
                                </Grid>
                                <Grid xs={12}>
                                    <SettingsMultiSelect
                                        label={t(
                                            'settings.general.kick_tags_label'
                                        )}
                                        tooltip={t(
                                            'settings.general.kick_tags_tooltip'
                                        )}
                                        newSettings={newSettings}
                                        setNewSettings={setNewSettings}
                                    />
                                </Grid>
                                <Grid xs={6}>
                                    <SettingsCheckBox
                                        label={t(
                                            'settings.general.party_warnings_enabled_label'
                                        )}
                                        tooltip={t(
                                            'settings.general.party_warnings_enabled_tooltip'
                                        )}
                                        enabled={
                                            newSettings.party_warnings_enabled
                                        }
                                        setEnabled={(
                                            party_warnings_enabled
                                        ) => {
                                            setNewSettings({
                                                ...newSettings,
                                                party_warnings_enabled
                                            });
                                        }}
                                    />
                                </Grid>
                                <Grid xs={6}>
                                    <SettingsCheckBox
                                        label={t(
                                            'settings.general.discord_presence_enabled_label'
                                        )}
                                        tooltip={t(
                                            'settings.general.discord_presence_enabled_tooltip'
                                        )}
                                        enabled={
                                            newSettings.discord_presence_enabled
                                        }
                                        setEnabled={(
                                            discord_presence_enabled
                                        ) => {
                                            setNewSettings({
                                                ...newSettings,
                                                discord_presence_enabled
                                            });
                                        }}
                                    />
                                </Grid>
                                <Grid xs={6}>
                                    <SettingsCheckBox
                                        label={t(
                                            'settings.general.auto_launch_game_label'
                                        )}
                                        tooltip={t(
                                            'settings.general.auto_launch_game_tooltip'
                                        )}
                                        enabled={newSettings.auto_launch_game}
                                        setEnabled={(auto_launch_game) => {
                                            setNewSettings({
                                                ...newSettings,
                                                auto_launch_game
                                            });
                                        }}
                                    />
                                </Grid>
                                <Grid xs={6}>
                                    <SettingsCheckBox
                                        label={t(
                                            'settings.general.auto_close_on_game_exit_label'
                                        )}
                                        tooltip={t(
                                            'settings.general.auto_close_on_game_exit_tooltip'
                                        )}
                                        enabled={
                                            newSettings.auto_close_on_game_exit
                                        }
                                        setEnabled={(
                                            auto_close_on_game_exit
                                        ) => {
                                            setNewSettings({
                                                ...newSettings,
                                                auto_close_on_game_exit
                                            });
                                        }}
                                    />
                                </Grid>

                                <Grid xs={6}>
                                    <SettingsCheckBox
                                        label={t(
                                            'settings.general.debug_log_enabled_label'
                                        )}
                                        tooltip={t(
                                            'settings.general.debug_log_enabled_tooltip'
                                        )}
                                        enabled={newSettings.debug_log_enabled}
                                        setEnabled={(debug_log_enabled) => {
                                            setNewSettings({
                                                ...newSettings,
                                                debug_log_enabled
                                            });
                                        }}
                                    />
                                </Grid>
                            </Grid>
                        </AccordionDetails>
                    </Accordion>

                    <Accordion
                        TransitionProps={{ unmountOnExit: true }}
                        expanded={expanded === 'lists'}
                        onChange={handleChange('lists')}
                    >
                        <AccordionSummary
                            expandIcon={<ExpandMoreIcon />}
                            aria-controls="lists-content"
                            id="lists-header"
                        >
                            <Typography sx={{ width: '33%', flexShrink: 0 }}>
                                <Trans
                                    i18nKey={'settings.player_lists.label'}
                                />
                            </Typography>
                            <Typography sx={{ color: 'text.secondary' }}>
                                <Trans
                                    i18nKey={
                                        'settings.player_lists.description'
                                    }
                                />
                            </Typography>
                        </AccordionSummary>
                        <AccordionDetails>
                            <Grid container>
                                {newSettings.lists.map((l: List, i: number) => {
                                    return (
                                        <Grid key={`list-row-${i}`} xs={12}>
                                            <Stack
                                                direction={'row'}
                                                spacing={1}
                                            >
                                                <IconButton
                                                    color={
                                                        l.enabled
                                                            ? 'primary'
                                                            : 'secondary'
                                                    }
                                                    onClick={() => {
                                                        toggleList(i);
                                                    }}
                                                >
                                                    {l.enabled ? (
                                                        <AlarmOnIcon />
                                                    ) : (
                                                        <AlarmOffIcon />
                                                    )}
                                                </IconButton>
                                                <IconButton
                                                    color={'warning'}
                                                    onClick={async () => {
                                                        await onOpenList(l, i);
                                                    }}
                                                >
                                                    <EditIcon />
                                                </IconButton>
                                                <IconButton
                                                    color={'error'}
                                                    onClick={() => {
                                                        deleteList(i);
                                                    }}
                                                >
                                                    <DeleteIcon />
                                                </IconButton>
                                                <Box
                                                    sx={{
                                                        display: 'flex',
                                                        alignItems: 'center'
                                                    }}
                                                >
                                                    <Typography
                                                        variant={'body1'}
                                                    >
                                                        {l.name}
                                                    </Typography>
                                                </Box>
                                            </Stack>
                                        </Grid>
                                    );
                                })}
                            </Grid>
                        </AccordionDetails>
                    </Accordion>

                    <Accordion
                        TransitionProps={{ unmountOnExit: true }}
                        expanded={expanded === 'links'}
                        onChange={handleChange('links')}
                    >
                        <AccordionSummary
                            expandIcon={<ExpandMoreIcon />}
                            aria-controls="links-content"
                            id="links-header"
                        >
                            <Typography sx={{ width: '33%', flexShrink: 0 }}>
                                <Trans
                                    i18nKey={'settings.external_links.label'}
                                />
                            </Typography>
                            <Typography sx={{ color: 'text.secondary' }}>
                                <Trans
                                    i18nKey={
                                        'settings.external_links.description'
                                    }
                                />
                            </Typography>
                        </AccordionSummary>
                        <AccordionDetails>
                            <Grid container>
                                {newSettings.links.map(
                                    (link: Link, i: number) => {
                                        return (
                                            <Grid key={`link-row-${i}`} xs={12}>
                                                <Stack
                                                    direction={'row'}
                                                    spacing={1}
                                                >
                                                    <IconButton
                                                        color={
                                                            link.enabled
                                                                ? 'primary'
                                                                : 'secondary'
                                                        }
                                                        onClick={() => {
                                                            toggleLink(i);
                                                        }}
                                                    >
                                                        {link.enabled ? (
                                                            <AlarmOnIcon />
                                                        ) : (
                                                            <AlarmOffIcon />
                                                        )}
                                                    </IconButton>
                                                    <IconButton
                                                        color={'warning'}
                                                        onClick={async () => {
                                                            await onOpenLink(
                                                                link,
                                                                i
                                                            );
                                                        }}
                                                    >
                                                        <EditIcon />
                                                    </IconButton>
                                                    <IconButton
                                                        color={'error'}
                                                        onClick={() => {
                                                            deleteLink(i);
                                                        }}
                                                    >
                                                        <DeleteIcon />
                                                    </IconButton>
                                                    <Box
                                                        sx={{
                                                            display: 'flex',
                                                            alignItems: 'center'
                                                        }}
                                                    >
                                                        <Typography
                                                            variant={'body1'}
                                                        >
                                                            {link.name}
                                                        </Typography>
                                                    </Box>
                                                </Stack>
                                            </Grid>
                                        );
                                    }
                                )}
                            </Grid>
                        </AccordionDetails>
                    </Accordion>

                    <Accordion
                        TransitionProps={{ unmountOnExit: true }}
                        expanded={expanded === 'http'}
                        onChange={handleChange('http')}
                    >
                        <AccordionSummary
                            expandIcon={<ExpandMoreIcon />}
                            aria-controls="http-content"
                            id="http-header"
                        >
                            <Typography sx={{ width: '33%', flexShrink: 0 }}>
                                <Trans i18nKey={'settings.http.label'} />
                            </Typography>
                            <Typography sx={{ color: 'text.secondary' }}>
                                <Trans i18nKey={'settings.http.description'} />
                            </Typography>
                        </AccordionSummary>
                        <AccordionDetails>
                            <Grid container>
                                <Grid xs={6}>
                                    <SettingsCheckBox
                                        label={t(
                                            'settings.http.http_enabled_label'
                                        )}
                                        tooltip={t(
                                            'settings.http.http_enabled_tooltip'
                                        )}
                                        enabled={newSettings.http_enabled}
                                        setEnabled={(http_enabled) => {
                                            setNewSettings({
                                                ...newSettings,
                                                http_enabled
                                            });
                                        }}
                                    />
                                </Grid>
                                <Grid xs={6}>
                                    <SettingsTextBox
                                        label={t(
                                            'settings.http.http_listen_addr_label'
                                        )}
                                        tooltip={t(
                                            'settings.http.http_listen_addr_tooltip'
                                        )}
                                        value={newSettings.http_listen_addr}
                                        setValue={(http_listen_addr) => {
                                            setNewSettings({
                                                ...newSettings,
                                                http_listen_addr
                                            });
                                        }}
                                        validator={validatorAddress}
                                    />
                                </Grid>
                                <Grid xs={12}>
                                    <Typography
                                        variant={'caption'}
                                        textAlign={'center'}
                                    >
                                        {t('settings.http.http_notice')}
                                    </Typography>
                                </Grid>
                            </Grid>
                        </AccordionDetails>
                    </Accordion>

                    <Accordion
                        TransitionProps={{ unmountOnExit: true }}
                        expanded={expanded === 'steam'}
                        onChange={handleChange('steam')}
                    >
                        <AccordionSummary
                            expandIcon={<ExpandMoreIcon />}
                            aria-controls="steam-content"
                            id="steam-header"
                        >
                            <Typography sx={{ width: '33%', flexShrink: 0 }}>
                                <Trans i18nKey={'settings.steam.label'} />
                            </Typography>
                            <Typography sx={{ color: 'text.secondary' }}>
                                <Trans i18nKey={'settings.steam.description'} />
                            </Typography>
                        </AccordionSummary>
                        <AccordionDetails>
                            <Grid container>
                                <Grid xs={6}>
                                    <SettingsTextBox
                                        label={t(
                                            'settings.steam.steam_id_label'
                                        )}
                                        tooltip={t(
                                            'settings.steam.steam_id_tooltip'
                                        )}
                                        value={newSettings.steam_id}
                                        setValue={(steam_id) => {
                                            setNewSettings({
                                                ...newSettings,
                                                steam_id
                                            });
                                        }}
                                        validator={validatorSteamID}
                                    />
                                </Grid>

                                <Grid xs={6}>
                                    <SettingsTextBox
                                        label={t(
                                            'settings.steam.api_key_label'
                                        )}
                                        tooltip={t(
                                            'settings.steam.api_key_tooltip'
                                        )}
                                        value={newSettings.api_key}
                                        secrets
                                        validator={makeValidatorLength(32)}
                                        setValue={(api_key) => {
                                            setNewSettings({
                                                ...newSettings,
                                                api_key
                                            });
                                        }}
                                    />
                                </Grid>
                                <Grid xs={12}>
                                    <SettingsTextBox
                                        label={t(
                                            'settings.steam.steam_dir_label'
                                        )}
                                        tooltip={t(
                                            'settings.steam.steam_dir_tooltip'
                                        )}
                                        value={newSettings.steam_dir}
                                        setValue={(steam_dir) => {
                                            setNewSettings({
                                                ...newSettings,
                                                steam_dir
                                            });
                                        }}
                                    />
                                </Grid>
                                <Grid xs={6}>
                                    <SettingsCheckBox
                                        label={t(
                                            'settings.steam.bd_api_enabled_label'
                                        )}
                                        tooltip={t(
                                            'settings.steam.bd_api_enabled_tooltip'
                                        )}
                                        enabled={newSettings.bd_api_enabled}
                                        setEnabled={(bd_api_enabled) => {
                                            setNewSettings({
                                                ...newSettings,
                                                bd_api_enabled
                                            });
                                        }}
                                    />
                                </Grid>
                                <Grid xs={6}>
                                    <SettingsTextBox
                                        disabled={!newSettings.bd_api_enabled}
                                        label={t(
                                            'settings.steam.bd_api_address_label'
                                        )}
                                        tooltip={t(
                                            'settings.steam.bd_api_address_tooltip'
                                        )}
                                        value={newSettings.bd_api_address}
                                        validator={(value) => {
                                            if (!isValidUrl(value)) {
                                                return 'Invalid URL';
                                            }
                                            return null;
                                        }}
                                        setValue={(bd_api_address: string) => {
                                            setNewSettings({
                                                ...newSettings,
                                                bd_api_address
                                            });
                                        }}
                                    />
                                </Grid>
                            </Grid>
                        </AccordionDetails>
                    </Accordion>

                    <Accordion
                        TransitionProps={{ unmountOnExit: true }}
                        expanded={expanded === 'tf2'}
                        onChange={handleChange('tf2')}
                    >
                        <AccordionSummary
                            expandIcon={<ExpandMoreIcon />}
                            aria-controls="tf2-content"
                            id="tf2-header"
                        >
                            <Typography sx={{ width: '33%', flexShrink: 0 }}>
                                <Trans i18nKey={'settings.tf2.label'} />
                            </Typography>
                            <Typography sx={{ color: 'text.secondary' }}>
                                <Trans i18nKey={'settings.tf2.label'} />
                            </Typography>
                        </AccordionSummary>
                        <AccordionDetails>
                            <Grid container>
                                <Grid xs={12}>
                                    <SettingsTextBox
                                        label={t('settings.tf2.tf2_dir_label')}
                                        tooltip={t(
                                            'settings.tf2.tf2_dir_tooltip'
                                        )}
                                        value={newSettings.tf2_dir}
                                        setValue={(tf2_dir) => {
                                            setNewSettings({
                                                ...newSettings,
                                                tf2_dir
                                            });
                                        }}
                                    />
                                </Grid>
                                <Grid xs={6}>
                                    <SettingsCheckBox
                                        label={t(
                                            'settings.tf2.rcon_static_label'
                                        )}
                                        tooltip={t(
                                            'settings.tf2.rcon_static_tooltip'
                                        )}
                                        enabled={newSettings.rcon_static}
                                        setEnabled={(rcon_static) => {
                                            setNewSettings({
                                                ...newSettings,
                                                rcon_static
                                            });
                                        }}
                                    />
                                </Grid>
                                <Grid xs={6}>
                                    <SettingsCheckBox
                                        label={t(
                                            'settings.tf2.voice_bans_enabled_label'
                                        )}
                                        tooltip={t(
                                            'settings.tf2.voice_bans_enabled_tooltip'
                                        )}
                                        enabled={newSettings.voice_bans_enabled}
                                        setEnabled={(voice_bans_enabled) => {
                                            setNewSettings({
                                                ...newSettings,
                                                voice_bans_enabled
                                            });
                                        }}
                                    />
                                </Grid>
                            </Grid>
                        </AccordionDetails>
                    </Accordion>
                </DialogContent>
                <DialogActions>
                    <CancelButton
                        onClick={async () => {
                            await settingsModal.hide();
                        }}
                    />
                    <ResetButton onClick={handleReset} />
                    <SaveButton onClick={onSubmit} />
                </DialogActions>
            </Dialog>
        );
    }
);

export default SettingsEditorModal;
