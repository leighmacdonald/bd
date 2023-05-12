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
import { Link, List, UserSettings } from '../api';
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

interface SettingsEditorProps {
    open: boolean;
    setOpen: (opeN: boolean) => void;
    origSettings: UserSettings;
}

export const SettingsEditor = ({
    open,
    setOpen,
    origSettings
}: SettingsEditorProps) => {
    const [listsOpen, setListsOpen] = useState(false);
    const [linksOpen, setLinksOpen] = useState(false);
    const [currentList, setCurrentList] = useState<List>();
    const [currentLink, setCurrentLink] = useState<Link>();

    const [settings, setSettings] = useState<UserSettings>(
        _.cloneDeep(origSettings)
    );

    const handleReset = useCallback(() => {
        setSettings(_.cloneDeep(origSettings));
    }, [origSettings]);

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
        console.log('Loaded in');
    }, [handleReset, origSettings]);

    const handleSave = useCallback(() => {
        setSettings(settings);
    }, [settings]);

    const handleClose = () => {
        setOpen(false);
    };

    // const onUpdateLists = useCallback(
    //     (lists: List[]) => {
    //         setSettings({ ...settings, lists });
    //     },
    //     [settings]
    // );

    const toggleList = (i: number) => {
        setSettings((s) => {
            s.lists[i].enabled = !s.lists[i].enabled;
            return s;
        });
    };

    const toggleLink = (i: number) => {
        setSettings((s) => {
            s.links[i].enabled = !s.links[i].enabled;
            return s;
        });
    };

    const deleteLink = (i: number) => {
        const newLinks = settings.links.filter((_, idx) => idx != i);
        setSettings({ ...settings, links: newLinks });
    };

    const deleteList = (i: number) => {
        const newList = settings.lists.filter((_, idx) => idx != i);
        setSettings({ ...settings, lists: newList });
    };

    const [expanded, setExpanded] = React.useState<string | false>('panel1');

    const handleChange =
        (panel: string) => (_: React.SyntheticEvent, newExpanded: boolean) => {
            setExpanded(newExpanded ? panel : false);
        };

    const theme = useTheme();

    return (
        <Dialog open={open} fullWidth>
            <DialogTitle component={Typography} variant={'h1'}>
                Settings
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
                            General
                        </Typography>
                        <Typography sx={{ color: 'text.secondary' }}>
                            Kicker, Tags, Chat Warnings
                        </Typography>
                    </AccordionSummary>
                    <AccordionDetails>
                        <Grid2 container spacing={1}>
                            <Grid2 xs={6}>
                                <SettingsCheckBox
                                    label={'Chat Warnings'}
                                    tooltip={
                                        'Enable in-game chat warnings to be broadcast to the active game'
                                    }
                                    enabled={settings.chat_warnings_enabled}
                                    setEnabled={(chat_warnings_enabled) => {
                                        setSettings({
                                            ...settings,
                                            chat_warnings_enabled
                                        });
                                    }}
                                />
                            </Grid2>
                            <Grid2 xs={6}>
                                <SettingsCheckBox
                                    label={'Kicker Enabled'}
                                    tooltip={
                                        'Enable the bot auto kick functionality when a match is found'
                                    }
                                    enabled={settings.kicker_enabled}
                                    setEnabled={(kicker_enabled) => {
                                        setSettings({
                                            ...settings,
                                            kicker_enabled
                                        });
                                    }}
                                />
                            </Grid2>
                            <Grid2 xs={12}>
                                <SettingsMultiSelect
                                    label={'Kickable Tag Matches'}
                                    tooltip={
                                        'Only matches which also match these tags will trigger a kick or notification.'
                                    }
                                    values={settings.kick_tags}
                                    setValues={(kick_tags) => {
                                        setSettings({ ...settings, kick_tags });
                                    }}
                                />
                            </Grid2>
                            <Grid2 xs={6}>
                                <SettingsCheckBox
                                    label={'Party Warnings Enabled'}
                                    tooltip={
                                        'Enable log messages to be broadcast to the lobby chat window'
                                    }
                                    enabled={settings.party_warnings_enabled}
                                    setEnabled={(party_warnings_enabled) => {
                                        setSettings({
                                            ...settings,
                                            party_warnings_enabled
                                        });
                                    }}
                                />
                            </Grid2>
                            <Grid2 xs={6}>
                                <SettingsCheckBox
                                    label={'Discord Presence Enabled'}
                                    tooltip={
                                        'Enable game status presence updates to your local discord client.'
                                    }
                                    enabled={settings.discord_presence_enabled}
                                    setEnabled={(discord_presence_enabled) => {
                                        setSettings({
                                            ...settings,
                                            discord_presence_enabled
                                        });
                                    }}
                                />
                            </Grid2>
                            <Grid2 xs={6}>
                                <SettingsCheckBox
                                    label={'Auto Launch Game On Start Up'}
                                    tooltip={
                                        'When enabled, upon launching bd, TF2 will also be launched at the same time'
                                    }
                                    enabled={settings.auto_launch_game}
                                    setEnabled={(auto_launch_game) => {
                                        setSettings({
                                            ...settings,
                                            auto_launch_game
                                        });
                                    }}
                                />
                            </Grid2>
                            <Grid2 xs={6}>
                                <SettingsCheckBox
                                    label={'Auto Close On Game Exit'}
                                    tooltip={
                                        'When enabled, upon the game existing, also shutdown bd.'
                                    }
                                    enabled={settings.auto_close_on_game_exit}
                                    setEnabled={(auto_close_on_game_exit) => {
                                        setSettings({
                                            ...settings,
                                            auto_close_on_game_exit
                                        });
                                    }}
                                />
                            </Grid2>

                            <Grid2 xs={6}>
                                <SettingsCheckBox
                                    label={'Enabled Debug Log'}
                                    tooltip={
                                        'When enabled, logs are written to bd.log in the application config root'
                                    }
                                    enabled={settings.debug_log_enabled}
                                    setEnabled={(debug_log_enabled) => {
                                        setSettings({
                                            ...settings,
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
                            Player &amp; Rules Lists
                        </Typography>
                        <Typography sx={{ color: 'text.secondary' }}>
                            Auth-Kicker, Tags, Chat Warnings
                        </Typography>
                    </AccordionSummary>
                    <AccordionDetails>
                        <Grid2 container>
                            {settings.lists.map((l, i) => {
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
                            External Links
                        </Typography>
                        <Typography sx={{ color: 'text.secondary' }}>
                            Configure custom menu links
                        </Typography>
                    </AccordionSummary>
                    <AccordionDetails>
                        <Grid2 container>
                            {settings.links.map((l, i) => {
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
                            HTTP Service
                        </Typography>
                        <Typography sx={{ color: 'text.secondary' }}>
                            Basic HTTP service settings
                        </Typography>
                    </AccordionSummary>
                    <AccordionDetails>
                        <Grid2 container>
                            <Grid2 xs={6}>
                                <SettingsCheckBox
                                    label={'Enable The HTTP Service*'}
                                    tooltip={
                                        'WARN: The HTTP service enabled the browser widget (this page) to function. You can only re-enable this ' +
                                        'service by editing the config file manually'
                                    }
                                    enabled={settings.http_enabled}
                                    setEnabled={(http_enabled) => {
                                        setSettings({
                                            ...settings,
                                            http_enabled
                                        });
                                    }}
                                />
                            </Grid2>
                            <Grid2 xs={6}>
                                <SettingsTextBox
                                    label={'Listen Address (host:port)'}
                                    tooltip={
                                        'What address the http service will listen on. (The URL you are connected to right now). You should use localhost' +
                                        'unless you know what you are doing as there is no authentication system.'
                                    }
                                    value={settings.http_listen_addr}
                                    setValue={(http_listen_addr) => {
                                        setSettings({
                                            ...settings,
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
                            Steam Config
                        </Typography>
                        <Typography sx={{ color: 'text.secondary' }}>
                            Configure steam api &amp; client
                        </Typography>
                    </AccordionSummary>
                    <AccordionDetails>
                        <Grid2 container>
                            <Grid2 xs={6}>
                                <SettingsTextBox
                                    label={'Steam ID'}
                                    value={settings.steam_id}
                                    setValue={(steam_id) => {
                                        setSettings({ ...settings, steam_id });
                                    }}
                                    validator={validatorSteamID}
                                    tooltip={
                                        'You can choose one of the following formats: steam,steam3,steam64'
                                    }
                                />
                            </Grid2>
                            <Grid2 xs={6}>
                                <SettingsTextBox
                                    label={'Steam API Key'}
                                    value={settings.api_key}
                                    secrets
                                    validator={makeValidatorLength(32)}
                                    setValue={(api_key) => {
                                        setSettings({ ...settings, api_key });
                                    }}
                                    tooltip={'Your personal steam web api key'}
                                />
                            </Grid2>
                            <Grid2 xs={12}>
                                <SettingsTextBox
                                    label={'Steam Root Directory'}
                                    value={settings.steam_dir}
                                    setValue={(steam_dir) => {
                                        setSettings({ ...settings, steam_dir });
                                    }}
                                    tooltip={
                                        'Location of your steam installation directory containing your userdata folder'
                                    }
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
                            TF2 Config
                        </Typography>
                        <Typography sx={{ color: 'text.secondary' }}>
                            Configure game settings
                        </Typography>
                    </AccordionSummary>
                    <AccordionDetails>
                        <Grid2 container>
                            <Grid2 xs={12}>
                                <SettingsTextBox
                                    label={'TF2 Root Directory'}
                                    value={settings.tf2_dir}
                                    setValue={(tf2_dir) => {
                                        setSettings({ ...settings, tf2_dir });
                                    }}
                                    tooltip={
                                        'Path to your steamapps/common/Team Fortress 2/tf` Folder'
                                    }
                                />
                            </Grid2>
                            <Grid2 xs={6}>
                                <SettingsCheckBox
                                    label={'RCON Static Mode'}
                                    tooltip={
                                        'When enabled, rcon will always use the static port and password of 21212 / pazer_sux_lol. Otherwise these are generated randomly on game launch'
                                    }
                                    enabled={settings.chat_warnings_enabled}
                                    setEnabled={(rcon_static) => {
                                        setSettings({
                                            ...settings,
                                            rcon_static
                                        });
                                    }}
                                />
                            </Grid2>
                            <Grid2 xs={6}>
                                <SettingsCheckBox
                                    label={'Generate Voice Bans'}
                                    tooltip={
                                        'WARN: This will overwrite your current ban list. Mutes the 200 most recent marked entries.'
                                    }
                                    enabled={settings.chat_warnings_enabled}
                                    setEnabled={(voice_bans_enabled) => {
                                        setSettings({
                                            ...settings,
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
                    onClick={handleClose}
                    startIcon={<CloseIcon />}
                    color={'error'}
                    variant={'contained'}
                >
                    Cancel
                </Button>
                <Button
                    onClick={handleSave}
                    startIcon={<SaveIcon />}
                    color={'success'}
                    variant={'contained'}
                >
                    Save
                </Button>
            </DialogActions>
        </Dialog>
    );
};
