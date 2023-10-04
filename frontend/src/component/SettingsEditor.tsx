import React, {
    ChangeEvent,
    SyntheticEvent,
    useCallback,
    useEffect,
    useMemo,
    useState
} from 'react';
import {
    Accordion,
    AccordionDetails,
    AccordionSummary,
    Autocomplete,
    Checkbox,
    DialogActions,
    DialogContent,
    DialogTitle,
    FormControl,
    FormControlLabel,
    FormGroup,
    FormHelperText,
    TextField,
    useTheme
} from '@mui/material';
import Dialog from '@mui/material/Dialog';
import Tooltip from '@mui/material/Tooltip';
import {
    Link,
    List,
    saveUserSettings,
    UserSettings,
    useUserSettings
} from '../api';
import cloneDeep from 'lodash/cloneDeep';
import Grid from '@mui/material/Unstable_Grid2';
import SteamID from 'steamid';
import Stack from '@mui/material/Stack';
import IconButton from '@mui/material/IconButton';
import AlarmOnIcon from '@mui/icons-material/AlarmOn';
import AlarmOffIcon from '@mui/icons-material/AlarmOff';
import EditIcon from '@mui/icons-material/Edit';
import DeleteIcon from '@mui/icons-material/Delete';
import Typography from '@mui/material/Typography';
import Box from '@mui/material/Box';
import AddIcon from '@mui/icons-material/Add';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import { Trans, useTranslation } from 'react-i18next';
import { logError } from '../util';
import NiceModal, { muiDialog, useModal } from '@ebay/nice-modal-react';
import {
    ModalSettingsAddKickTag,
    ModalSettingsLinks,
    ModalSettingsList
} from '../modals';
import { CancelButton, ResetButton, SaveButton } from './Buttons';
import { sortedUniq } from 'lodash';

export type inputValidator = (value: string) => string | null;

interface SettingsTextBoxProps {
    label: string;
    value: string;
    setValue: (value: string) => void;
    tooltip: string;
    secrets?: boolean;
    validator?: inputValidator;
}

export const SettingsTextBox = ({
    label,
    value,
    setValue,
    tooltip,
    secrets,
    validator
}: SettingsTextBoxProps) => {
    const [error, setError] = useState<string | null>(null);
    const handleChange = useCallback(
        (event: ChangeEvent<HTMLTextAreaElement>) => {
            setValue(event.target.value);
            if (validator) {
                setError(validator(event.target.value));
            }
        },
        [setValue, validator]
    );

    return (
        <Tooltip title={tooltip} placement={'top'}>
            <FormControl fullWidth>
                <TextField
                    hiddenLabel
                    type={secrets ? 'password' : 'text'}
                    error={Boolean(error)}
                    id={`settings-textfield-${label}`}
                    label={label}
                    value={value}
                    onChange={handleChange}
                />
                <FormHelperText>{error}</FormHelperText>
            </FormControl>
        </Tooltip>
    );
};

const validatorSteamID = (value: string): string => {
    let err = 'Invalid SteamID';
    try {
        const id = new SteamID(value);
        if (id.isValid()) {
            err = '';
        }
    } catch (_) {
        /* empty */
    }
    return err;
};

const makeValidatorLength = (length: number): inputValidator => {
    return (value: string): string => {
        if (value.length != length) {
            return 'Invalid value';
        }
        return '';
    };
};

export const isStringIp = (value: string): boolean => {
    return /^(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$/.test(
        value
    );
};

const validatorAddress = (value: string): string => {
    const pcs = value.split(':');
    if (pcs.length != 2) {
        return 'Format must match host:port';
    }
    if (pcs[0].toLowerCase() != 'localhost' && !isStringIp(pcs[0])) {
        return 'Invalid address. x.x.x.x or localhost accepted';
    }
    const port = parseInt(pcs[1], 10);
    if (!/^\d+$/.test(pcs[1])) {
        return 'Invalid port, must be positive integer';
    }
    if (port <= 0 || port > 65535) {
        return 'Invalid port, must be in range: 1-65535';
    }
    return '';
};

interface SettingsCheckBoxProps {
    label: string;
    enabled: boolean;
    setEnabled: (checked: boolean) => void;
    tooltip: string;
}

export const SettingsCheckBox = ({
    label,
    enabled,
    setEnabled,
    tooltip
}: SettingsCheckBoxProps) => {
    return (
        <FormGroup>
            <Tooltip title={tooltip} placement="top">
                <FormControlLabel
                    control={
                        <Checkbox
                            checked={enabled}
                            onChange={(_, checked) => {
                                setEnabled(checked);
                            }}
                        />
                    }
                    label={label}
                />
            </Tooltip>
        </FormGroup>
    );
};

interface SettingsMultiSelectProps {
    label: string;
    newSettings: UserSettings;
    setNewSettings: React.Dispatch<React.SetStateAction<UserSettings>>;
    tooltip: string;
}

export const SettingsMultiSelect = ({
    newSettings,
    setNewSettings,
    label,
    tooltip
}: SettingsMultiSelectProps) => {
    const modal = useModal();

    const onAddKickTag = useCallback(async () => {
        try {
            await NiceModal.show(ModalSettingsAddKickTag, { setNewSettings });
        } catch (e) {
            logError(e);
        } finally {
            await modal.hide();
        }
    }, [modal, setNewSettings]);

    const handleChange = (
        _: SyntheticEvent<Element, Event>,
        value: string | string[]
    ) => {
        setNewSettings((prevState) => {
            const tags = sortedUniq([
                ...(typeof value === 'string' ? value.split(',') : value)
            ]);
            return {
                ...prevState,
                kick_tags: tags
            };
        });
    };

    const validTags = useMemo(() => {
        return sortedUniq([
            ...newSettings.unique_tags,
            ...newSettings.kick_tags
        ]);
    }, [newSettings.unique_tags, newSettings.kick_tags]);

    return (
        <Stack direction={'row'} spacing={1}>
            <Tooltip title={tooltip} placement="top">
                <FormControl fullWidth>
                    <Autocomplete
                        multiple
                        id="kick_tags-select"
                        value={newSettings.kick_tags}
                        onChange={handleChange}
                        //getOptionLabel={(option) => option.title}
                        renderInput={(params) => (
                            <TextField
                                {...params}
                                variant={'outlined'}
                                label={label}
                                placeholder="Tags"
                            />
                        )}
                        options={validTags}
                    />
                </FormControl>
            </Tooltip>
            <Box sx={{ display: 'flex', alignItems: 'center' }}>
                <IconButton color={'success'} onClick={onAddKickTag}>
                    <AddIcon />
                </IconButton>
            </Box>
        </Stack>
    );
};

export const SettingsEditor = NiceModal.create(() => {
    const { settings, setSettings } = useUserSettings();
    const modal = useModal();
    const { t } = useTranslation();
    const theme = useTheme();

    const [newSettings, setNewSettings] = useState<UserSettings>(
        cloneDeep(settings)
    );

    const handleReset = useCallback(() => {
        setNewSettings(cloneDeep(settings));
    }, [settings, setNewSettings]);

    const onOpenLink = useCallback(
        async (link: Link, rowIndex: number) => {
            try {
                await NiceModal.show(ModalSettingsLinks, {
                    link,
                    rowIndex,
                    setNewSettings
                });
            } catch (e) {
                logError(e);
            } finally {
                console.log(newSettings.links);
                await modal.hide();
            }
        },
        [modal, newSettings.links]
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
                await modal.hide();
            }
        },
        [modal, newSettings.lists]
    );

    useEffect(() => {
        handleReset();
    }, [handleReset]);

    const handleSave = useCallback(async () => {
        try {
            await saveUserSettings(newSettings);
            setSettings(newSettings);
        } catch (reason) {
            logError(reason);
        } finally {
            await modal.hide();
        }
    }, [newSettings, modal, setSettings]);

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

    const [expanded, setExpanded] = React.useState<string | false>('general');

    const handleChange =
        (panel: string) => (_: React.SyntheticEvent, newExpanded: boolean) => {
            setExpanded(newExpanded ? panel : false);
        };

    return (
        <Dialog fullWidth {...muiDialog(modal)}>
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
                                    enabled={newSettings.chat_warnings_enabled}
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
                                    enabled={newSettings.party_warnings_enabled}
                                    setEnabled={(party_warnings_enabled) => {
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
                                    setEnabled={(discord_presence_enabled) => {
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
                                    setEnabled={(auto_close_on_game_exit) => {
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
                            <Trans i18nKey={'settings.player_lists.label'} />
                        </Typography>
                        <Typography sx={{ color: 'text.secondary' }}>
                            <Trans
                                i18nKey={'settings.player_lists.description'}
                            />
                        </Typography>
                    </AccordionSummary>
                    <AccordionDetails>
                        <Grid container>
                            {newSettings.lists.map((l: List, i: number) => {
                                return (
                                    <Grid key={`list-row-${i}`} xs={12}>
                                        <Stack direction={'row'} spacing={1}>
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
                                                <Typography variant={'body1'}>
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
                            <Trans i18nKey={'settings.external_links.label'} />
                        </Typography>
                        <Typography sx={{ color: 'text.secondary' }}>
                            <Trans
                                i18nKey={'settings.external_links.description'}
                            />
                        </Typography>
                    </AccordionSummary>
                    <AccordionDetails>
                        <Grid container>
                            {newSettings.links.map((link: Link, i: number) => {
                                return (
                                    <Grid key={`link-row-${i}`} xs={12}>
                                        <Stack direction={'row'} spacing={1}>
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
                                                    await onOpenLink(link, i);
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
                                                <Typography variant={'body1'}>
                                                    {link.name}
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
                                    label={t('settings.steam.steam_id_label')}
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
                                    label={t('settings.steam.api_key_label')}
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
                                    label={t('settings.steam.steam_dir_label')}
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
                                    tooltip={t('settings.tf2.tf2_dir_tooltip')}
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
                                    label={t('settings.tf2.rcon_static_label')}
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
                <CancelButton />
                <ResetButton onClick={handleReset} />
                <SaveButton onClick={handleSave} />
            </DialogActions>
        </Dialog>
    );
});
