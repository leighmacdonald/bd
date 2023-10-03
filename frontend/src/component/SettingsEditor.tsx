import React, {
    ChangeEvent,
    useCallback,
    useContext,
    useEffect,
    useState
} from 'react';
import {
    Accordion,
    AccordionDetails,
    AccordionSummary,
    Button,
    Checkbox,
    DialogActions,
    DialogContent,
    DialogTitle,
    FormControl,
    FormControlLabel,
    FormGroup,
    FormHelperText,
    InputLabel,
    ListItemText,
    OutlinedInput,
    Select,
    SelectChangeEvent,
    TextField,
    useTheme
} from '@mui/material';
import Dialog from '@mui/material/Dialog';
import Tooltip from '@mui/material/Tooltip';
import CloseIcon from '@mui/icons-material/Close';
import MenuItem from '@mui/material/MenuItem';
import {
    Link,
    List,
    saveUserSettings,
    UserSettings,
    useUserSettings
} from '../api';
import _ from 'lodash';
import { SettingsContext } from '../context/settings';
import Grid2 from '@mui/material/Unstable_Grid2';
import SteamID from 'steamid';
import Stack from '@mui/material/Stack';
import IconButton from '@mui/material/IconButton';
import AlarmOnIcon from '@mui/icons-material/AlarmOn';
import AlarmOffIcon from '@mui/icons-material/AlarmOff';
import EditIcon from '@mui/icons-material/Edit';
import DeleteIcon from '@mui/icons-material/Delete';
import Typography from '@mui/material/Typography';
import Box from '@mui/material/Box';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import SaveIcon from '@mui/icons-material/Save';
import { SettingsListEditor } from './SettingsListEditor';
import { SettingsLinkEditor } from './SettingsLinkEditor';
import { Trans, useTranslation } from 'react-i18next';
import { logError } from '../util';
import NiceModal, { muiDialog, useModal } from '@ebay/nice-modal-react';

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
    values: string[];
    setValues: (values: string[]) => void;
    tooltip: string;
}

export const SettingsMultiSelect = ({
    values,
    setValues,
    label,
    tooltip
}: SettingsMultiSelectProps) => {
    const { settings } = useContext(SettingsContext);
    const handleChange = (event: SelectChangeEvent<typeof values>) => {
        const {
            target: { value }
        } = event;
        setValues(typeof value === 'string' ? value.split(',') : value);
    };

    return (
        <Tooltip title={tooltip} placement="top">
            <FormControl fullWidth>
                <InputLabel id={`settings-select-${label}-label`}>
                    {label}
                </InputLabel>
                <Select<string[]>
                    fullWidth
                    labelId={`settings-select-${label}-label`}
                    id={`settings-select-${label}`}
                    multiple
                    value={values}
                    defaultValue={values}
                    onChange={handleChange}
                    input={<OutlinedInput label="Tag" />}
                    renderValue={(selected) => selected.join(', ')}
                >
                    {settings.kick_tags.map((name) => (
                        <MenuItem key={name} value={name}>
                            <Checkbox checked={values.indexOf(name) > -1} />
                            <ListItemText primary={name} />
                        </MenuItem>
                    ))}
                </Select>
            </FormControl>
        </Tooltip>
    );
};

export const SettingsEditor = NiceModal.create(() => {
    const [listsOpen, setListsOpen] = useState(false);
    const [linksOpen, setLinksOpen] = useState(false);
    const [currentList, setCurrentList] = useState<List>();
    const [currentLink, setCurrentLink] = useState<Link>();

    const modal = useModal();
    const { settings, setSettings } = useUserSettings();
    const { t } = useTranslation();

    const [newSettings, setNewSettings] = useState<UserSettings>(
        _.cloneDeep(settings)
    );

    const handleReset = useCallback(() => {
        setNewSettings(_.cloneDeep(settings));
    }, [settings, setNewSettings]);

    const onOpenList = (list: List) => {
        setCurrentList(list);
        setListsOpen(true);
    };

    const onOpenLink = (link: Link) => {
        setCurrentLink(link);
        setLinksOpen(true);
    };

    useEffect(() => {
        handleReset();
    }, [handleReset, settings]);

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

    const toggleList = (i: number) => {
        setNewSettings((s: UserSettings) => {
            s.lists[i].enabled = !s.lists[i].enabled;
            return s;
        });
    };

    const toggleLink = (i: number) => {
        setNewSettings((s: UserSettings) => {
            s.links[i].enabled = !s.links[i].enabled;
            return s;
        });
    };

    const deleteLink = (i: number) => {
        const newLinks = newSettings.links.filter((_, idx) => idx != i);
        setNewSettings({ ...newSettings, links: newLinks });
    };

    const deleteList = (i: number) => {
        const newList = newSettings.lists.filter((_, idx) => idx != i);
        setNewSettings({ ...newSettings, lists: newList });
    };

    const [expanded, setExpanded] = React.useState<string | false>('panel1');

    const handleChange =
        (panel: string) => (_: React.SyntheticEvent, newExpanded: boolean) => {
            setExpanded(newExpanded ? panel : false);
        };

    const theme = useTheme();

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
                        <Grid2 container spacing={1}>
                            <Grid2 xs={6}>
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
                            </Grid2>
                            <Grid2 xs={6}>
                                <SettingsCheckBox
                                    label={t(
                                        'settings.general.kicker_enabled_label'
                                    )}
                                    tooltip={t(
                                        'settings.general.kicker_enabled_description'
                                    )}
                                    enabled={newSettings.kicker_enabled}
                                    setEnabled={(kicker_enabled) => {
                                        setNewSettings({
                                            ...newSettings,
                                            kicker_enabled
                                        });
                                    }}
                                />
                            </Grid2>
                            <Grid2 xs={12}>
                                <SettingsMultiSelect
                                    label={t(
                                        'settings.general.kick_tags_label'
                                    )}
                                    tooltip={t(
                                        'settings.general.kick_tags_description'
                                    )}
                                    values={newSettings.kick_tags}
                                    setValues={(kick_tags) => {
                                        setNewSettings({
                                            ...newSettings,
                                            kick_tags
                                        });
                                    }}
                                />
                            </Grid2>
                            <Grid2 xs={6}>
                                <SettingsCheckBox
                                    label={t(
                                        'settings.general.party_warnings_enabled_label'
                                    )}
                                    tooltip={t(
                                        'settings.general.party_warnings_enabled_description'
                                    )}
                                    enabled={newSettings.party_warnings_enabled}
                                    setEnabled={(party_warnings_enabled) => {
                                        setNewSettings({
                                            ...newSettings,
                                            party_warnings_enabled
                                        });
                                    }}
                                />
                            </Grid2>
                            <Grid2 xs={6}>
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
                            </Grid2>
                            <Grid2 xs={6}>
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
                            </Grid2>
                            <Grid2 xs={6}>
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
                            </Grid2>

                            <Grid2 xs={6}>
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
                            </Grid2>
                        </Grid2>
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
                        <Grid2 container>
                            {newSettings.lists.map((l, i) => {
                                return (
                                    <Grid2 key={`list-row-${i}`} xs={12}>
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
                                                color={'primary'}
                                                onClick={() => {
                                                    onOpenList(l);
                                                }}
                                            >
                                                <EditIcon />
                                            </IconButton>
                                            <IconButton
                                                color={'primary'}
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
                                    </Grid2>
                                );
                            })}
                        </Grid2>
                        {currentList && (
                            <SettingsListEditor
                                open={listsOpen}
                                value={currentList}
                                setValue={setCurrentList}
                                setOpen={setListsOpen}
                                isNew={currentList.name == ''}
                            />
                        )}
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
                        <Grid2 container>
                            {newSettings.links.map((l, i) => {
                                return (
                                    <Grid2 key={`list-row-${i}`} xs={12}>
                                        <Stack direction={'row'} spacing={1}>
                                            <IconButton
                                                color={
                                                    l.enabled
                                                        ? 'primary'
                                                        : 'secondary'
                                                }
                                                onClick={() => {
                                                    toggleLink(i);
                                                }}
                                            >
                                                {l.enabled ? (
                                                    <AlarmOnIcon />
                                                ) : (
                                                    <AlarmOffIcon />
                                                )}
                                            </IconButton>
                                            <IconButton
                                                color={'primary'}
                                                onClick={() => {
                                                    onOpenLink(l);
                                                }}
                                            >
                                                <EditIcon />
                                            </IconButton>
                                            <IconButton
                                                color={'primary'}
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
                                                    {l.name}
                                                </Typography>
                                            </Box>
                                        </Stack>
                                    </Grid2>
                                );
                            })}
                        </Grid2>
                        {currentLink && (
                            <SettingsLinkEditor
                                open={linksOpen}
                                value={currentLink}
                                setValue={setCurrentLink}
                                setOpen={setLinksOpen}
                                isNew={currentLink.name == ''}
                            />
                        )}
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
                        <Grid2 container>
                            <Grid2 xs={6}>
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
                            </Grid2>
                            <Grid2 xs={6}>
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
                            </Grid2>
                        </Grid2>
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
                        <Grid2 container>
                            <Grid2 xs={6}>
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
                            </Grid2>
                            <Grid2 xs={6}>
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
                            </Grid2>
                            <Grid2 xs={12}>
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
                            </Grid2>
                        </Grid2>
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
                        <Grid2 container>
                            <Grid2 xs={12}>
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
                            </Grid2>
                            <Grid2 xs={6}>
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
                            </Grid2>
                            <Grid2 xs={6}>
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
                            </Grid2>
                        </Grid2>
                    </AccordionDetails>
                </Accordion>
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
                    startIcon={<CloseIcon />}
                    color={'error'}
                    variant={'contained'}
                >
                    <Trans i18nKey={'button.reset'} />
                </Button>
                <Button
                    onClick={handleSave}
                    startIcon={<SaveIcon />}
                    color={'success'}
                    variant={'contained'}
                >
                    <Trans i18nKey={'button.save'} />
                </Button>
            </DialogActions>
        </Dialog>
    );
});
