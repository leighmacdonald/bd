import { createFileRoute } from '@tanstack/react-router';
import {
    Accordion,
    AccordionDetails,
    AccordionSummary,
    useTheme
} from '@mui/material';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { Trans, useTranslation } from 'react-i18next';
import { Link, List, saveUserSettingsMutation, UserSettings } from '../api.ts';
import Stack from '@mui/material/Stack';
import IconButton from '@mui/material/IconButton';
import AlarmOnIcon from '@mui/icons-material/AlarmOn';
import AlarmOffIcon from '@mui/icons-material/AlarmOff';
import EditIcon from '@mui/icons-material/Edit';
import DeleteIcon from '@mui/icons-material/Delete';
import Box from '@mui/material/Box';
import { logError } from '../util.ts';
import NiceModal from '@ebay/nice-modal-react';
import { ModalSettingsList } from '../component/modal';
import { SyntheticEvent, useCallback, useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { useSettingsState } from '../context/SettingsContext.ts';
import { useForm } from '@tanstack/react-form';
import { zodValidator } from '@tanstack/zod-form-adapter';
import { CheckboxSimple } from '../component/fields/CheckboxSimple.tsx';
import { z } from 'zod';
import { Buttons } from '../component/fields/Buttons.tsx';
import { TextFieldSimple } from '../component/fields/TextFieldSimple.tsx';
import { SelectFieldSimple } from '../component/fields/SelectFieldSimple.tsx';
import MenuItem from '@mui/material/MenuItem';

export const Route = createFileRoute('/settings')({
    component: Settings
});

function Settings() {
    const { t } = useTranslation();
    const theme = useTheme();
    const { settings, setSettings } = useSettingsState();

    const onOpenList = useCallback(
        async (list: List, rowIndex: number) => {
            try {
                await NiceModal.show(ModalSettingsList, {
                    list,
                    rowIndex,
                    setSettings
                });
            } catch (e) {
                logError(e);
            } finally {
                console.log(settings.lists);
            }
        },
        [settings.lists]
    );

    const options = saveUserSettingsMutation();

    const settingsMutation = useMutation({
        ...options,
        onSuccess: async () => {
            console.log('settings saved');
        },
        onError: (error) => {
            console.log(`settings save error: ${error}`);
        }
    });

    const toggleList = useCallback(
        (i: number) => {
            setSettings((us: UserSettings) => {
                const s = { ...us };
                s.lists[i].enabled = !s.lists[i].enabled;
                return s;
            });
        },
        [setSettings]
    );

    const toggleLink = useCallback(
        (i: number) => {
            setSettings((us: UserSettings) => {
                const s = { ...us };
                s.links[i].enabled = !s.links[i].enabled;
                return s;
            });
        },
        [setSettings]
    );

    const deleteLink = useCallback(
        (i: number) => {
            const newLinks = settings.links.filter(
                (_: Link, idx: number) => idx != i
            );
            setSettings({ ...settings, links: newLinks });
        },
        [settings, setSettings]
    );

    const deleteList = useCallback(
        (i: number) => {
            const newList = settings.lists.filter(
                (_: List, idx: number) => idx != i
            );
            setSettings({ ...settings, lists: newList });
        },
        [settings, setSettings]
    );

    const [expanded, setExpanded] = useState<string | false>('general');

    const handleChange =
        (panel: string) => (_: SyntheticEvent, newExpanded: boolean) => {
            setExpanded(newExpanded ? panel : false);
        };

    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            settingsMutation.mutate(value);
        },
        validatorAdapter: zodValidator,
        defaultValues: settings
    });

    return (
        <Grid container>
            <Grid xs={12} padding={2}>
                <form
                    onSubmit={async (e) => {
                        e.preventDefault();
                        e.stopPropagation();
                        await handleSubmit();
                    }}
                >
                    <Grid container>
                        <Grid xs={12}>
                            <Accordion
                                expanded={expanded === 'general'}
                                onChange={handleChange('general')}
                                slotProps={{
                                    transition: { unmountOnExit: true }
                                }}
                            >
                                <AccordionSummary
                                    style={{
                                        backgroundColor:
                                            theme.palette.background.paper
                                    }}
                                    expandIcon={<ExpandMoreIcon />}
                                    aria-controls="general-content"
                                    id="general-header"
                                >
                                    <Typography
                                        sx={{ width: '33%', flexShrink: 0 }}
                                    >
                                        {t('settings.general.label')}
                                    </Typography>
                                    <Typography
                                        sx={{ color: 'text.secondary' }}
                                    >
                                        {t('settings.general.description')}
                                    </Typography>
                                </AccordionSummary>
                                <AccordionDetails>
                                    <Grid container spacing={1}>
                                        <Grid xs={6}>
                                            <Field
                                                name={'chat_warnings_enabled'}
                                                validators={{
                                                    onSubmit: z.boolean()
                                                }}
                                                children={(props) => {
                                                    return (
                                                        <CheckboxSimple
                                                            {...props}
                                                            label={t(
                                                                'settings.general.chat_warnings_label'
                                                            )}
                                                            tooltip={t(
                                                                'settings.general.chat_warnings_tooltip'
                                                            )}
                                                        />
                                                    );
                                                }}
                                            />
                                        </Grid>
                                        <Grid xs={6}>
                                            <Field
                                                name={'kicker_enabled'}
                                                validators={{
                                                    onSubmit: z.boolean()
                                                }}
                                                children={(props) => {
                                                    return (
                                                        <CheckboxSimple
                                                            {...props}
                                                            label={t(
                                                                'settings.general.kicker_enabled_label'
                                                            )}
                                                            tooltip={t(
                                                                'settings.general.kicker_enabled_tooltip'
                                                            )}
                                                        />
                                                    );
                                                }}
                                            />
                                        </Grid>
                                        <Grid xs={12}>
                                            <Field
                                                name={'kick_tags'}
                                                validators={{
                                                    onSubmit: z.string()
                                                }}
                                                children={(props) => {
                                                    return (
                                                        <SelectFieldSimple
                                                            {...props}
                                                            label={'Action'}
                                                            items={[
                                                                settings.kick_tags
                                                            ]}
                                                            renderMenu={(
                                                                fa
                                                            ) => {
                                                                return (
                                                                    <MenuItem
                                                                        value={
                                                                            fa
                                                                        }
                                                                        key={`fa-${fa}`}
                                                                    >
                                                                        {fa}
                                                                    </MenuItem>
                                                                );
                                                            }}
                                                        />
                                                    );
                                                }}
                                            />
                                        </Grid>
                                        <Grid xs={6}>
                                            <Field
                                                name={'party_warnings_enabled'}
                                                validators={{
                                                    onSubmit: z.boolean()
                                                }}
                                                children={(props) => {
                                                    return (
                                                        <CheckboxSimple
                                                            {...props}
                                                            label={t(
                                                                'settings.general.party_warnings_enabled_label'
                                                            )}
                                                            tooltip={t(
                                                                'settings.general.party_warnings_enabled_tooltip'
                                                            )}
                                                        />
                                                    );
                                                }}
                                            />
                                        </Grid>
                                        <Grid xs={6}>
                                            <Field
                                                name={
                                                    'discord_presence_enabled'
                                                }
                                                validators={{
                                                    onSubmit: z.boolean()
                                                }}
                                                children={(props) => {
                                                    return (
                                                        <CheckboxSimple
                                                            {...props}
                                                            label={t(
                                                                'settings.general.discord_presence_enabled_label'
                                                            )}
                                                            tooltip={t(
                                                                'settings.general.discord_presence_enabled_tooltip'
                                                            )}
                                                        />
                                                    );
                                                }}
                                            />
                                        </Grid>
                                        <Grid xs={6}>
                                            <Field
                                                name={'auto_launch_game'}
                                                validators={{
                                                    onSubmit: z.boolean()
                                                }}
                                                children={(props) => {
                                                    return (
                                                        <CheckboxSimple
                                                            {...props}
                                                            label={t(
                                                                'settings.general.auto_launch_game_label'
                                                            )}
                                                            tooltip={t(
                                                                'settings.general.auto_launch_game_tooltip'
                                                            )}
                                                        />
                                                    );
                                                }}
                                            />
                                        </Grid>
                                        <Grid xs={6}>
                                            <Field
                                                name={'auto_close_on_game_exit'}
                                                validators={{
                                                    onSubmit: z.boolean()
                                                }}
                                                children={(props) => {
                                                    return (
                                                        <CheckboxSimple
                                                            {...props}
                                                            label={t(
                                                                'settings.general.auto_close_on_game_exit_label'
                                                            )}
                                                            tooltip={t(
                                                                'settings.general.auto_close_on_game_exit_tooltip'
                                                            )}
                                                        />
                                                    );
                                                }}
                                            />
                                        </Grid>

                                        <Grid xs={6}>
                                            <Field
                                                name={'debug_log_enabled'}
                                                validators={{
                                                    onSubmit: z.boolean()
                                                }}
                                                children={(props) => {
                                                    return (
                                                        <CheckboxSimple
                                                            {...props}
                                                            label={t(
                                                                'settings.general.debug_log_enabled_label'
                                                            )}
                                                            tooltip={t(
                                                                'settings.general.debug_log_enabled_tooltip'
                                                            )}
                                                        />
                                                    );
                                                }}
                                            />
                                        </Grid>
                                    </Grid>
                                </AccordionDetails>
                            </Accordion>

                            <Accordion
                                slotProps={{
                                    transition: { unmountOnExit: true }
                                }}
                                expanded={expanded === 'lists'}
                                onChange={handleChange('lists')}
                            >
                                <AccordionSummary
                                    expandIcon={<ExpandMoreIcon />}
                                    aria-controls="lists-content"
                                    id="lists-header"
                                >
                                    <Typography
                                        sx={{ width: '33%', flexShrink: 0 }}
                                    >
                                        <Trans
                                            i18nKey={
                                                'settings.player_lists.label'
                                            }
                                        />
                                    </Typography>
                                    <Typography
                                        sx={{ color: 'text.secondary' }}
                                    >
                                        <Trans
                                            i18nKey={
                                                'settings.player_lists.description'
                                            }
                                        />
                                    </Typography>
                                </AccordionSummary>
                                <AccordionDetails>
                                    <Grid container>
                                        {settings.lists.map(
                                            (l: List, i: number) => {
                                                return (
                                                    <Grid
                                                        key={`list-row-${i}`}
                                                        xs={12}
                                                    >
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
                                                                    toggleList(
                                                                        i
                                                                    );
                                                                }}
                                                            >
                                                                {l.enabled ? (
                                                                    <AlarmOnIcon />
                                                                ) : (
                                                                    <AlarmOffIcon />
                                                                )}
                                                            </IconButton>
                                                            <IconButton
                                                                color={
                                                                    'warning'
                                                                }
                                                                onClick={async () => {
                                                                    await onOpenList(
                                                                        l,
                                                                        i
                                                                    );
                                                                }}
                                                            >
                                                                <EditIcon />
                                                            </IconButton>
                                                            <IconButton
                                                                color={'error'}
                                                                onClick={() => {
                                                                    deleteList(
                                                                        i
                                                                    );
                                                                }}
                                                            >
                                                                <DeleteIcon />
                                                            </IconButton>
                                                            <Box
                                                                sx={{
                                                                    display:
                                                                        'flex',
                                                                    alignItems:
                                                                        'center'
                                                                }}
                                                            >
                                                                <Typography
                                                                    variant={
                                                                        'body1'
                                                                    }
                                                                >
                                                                    {l.name}
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
                                slotProps={{
                                    transition: { unmountOnExit: true }
                                }}
                                expanded={expanded === 'links'}
                                onChange={handleChange('links')}
                            >
                                <AccordionSummary
                                    expandIcon={<ExpandMoreIcon />}
                                    aria-controls="links-content"
                                    id="links-header"
                                >
                                    <Typography
                                        sx={{ width: '33%', flexShrink: 0 }}
                                    >
                                        <Trans
                                            i18nKey={
                                                'settings.external_links.label'
                                            }
                                        />
                                    </Typography>
                                    <Typography
                                        sx={{ color: 'text.secondary' }}
                                    >
                                        <Trans
                                            i18nKey={
                                                'settings.external_links.description'
                                            }
                                        />
                                    </Typography>
                                </AccordionSummary>
                                <AccordionDetails>
                                    <Grid container>
                                        {settings.links.map(
                                            (link: Link, i: number) => {
                                                return (
                                                    <Grid
                                                        key={`link-row-${i}`}
                                                        xs={12}
                                                    >
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
                                                                    toggleLink(
                                                                        i
                                                                    );
                                                                }}
                                                            >
                                                                {link.enabled ? (
                                                                    <AlarmOnIcon />
                                                                ) : (
                                                                    <AlarmOffIcon />
                                                                )}
                                                            </IconButton>
                                                            <IconButton
                                                                color={
                                                                    'warning'
                                                                }
                                                            >
                                                                <EditIcon />
                                                            </IconButton>
                                                            <IconButton
                                                                color={'error'}
                                                                onClick={() => {
                                                                    deleteLink(
                                                                        i
                                                                    );
                                                                }}
                                                            >
                                                                <DeleteIcon />
                                                            </IconButton>
                                                            <Box
                                                                sx={{
                                                                    display:
                                                                        'flex',
                                                                    alignItems:
                                                                        'center'
                                                                }}
                                                            >
                                                                <Typography
                                                                    variant={
                                                                        'body1'
                                                                    }
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
                                expanded={expanded === 'http'}
                                onChange={handleChange('http')}
                                slotProps={{
                                    transition: { unmountOnExit: true }
                                }}
                            >
                                <AccordionSummary
                                    expandIcon={<ExpandMoreIcon />}
                                    aria-controls="http-content"
                                    id="http-header"
                                >
                                    <Typography
                                        sx={{ width: '33%', flexShrink: 0 }}
                                    >
                                        <Trans
                                            i18nKey={'settings.http.label'}
                                        />
                                    </Typography>
                                    <Typography
                                        sx={{ color: 'text.secondary' }}
                                    >
                                        <Trans
                                            i18nKey={
                                                'settings.http.description'
                                            }
                                        />
                                    </Typography>
                                </AccordionSummary>
                                <AccordionDetails>
                                    <Grid container>
                                        <Grid xs={6}>
                                            <Field
                                                name={'http_enabled'}
                                                validators={{
                                                    onSubmit: z.boolean()
                                                }}
                                                children={(props) => {
                                                    return (
                                                        <CheckboxSimple
                                                            {...props}
                                                            label={t(
                                                                'settings.http.http_enabled_label'
                                                            )}
                                                            tooltip={t(
                                                                'settings.http.http_enabled_tooltip'
                                                            )}
                                                        />
                                                    );
                                                }}
                                            />
                                        </Grid>
                                        <Grid xs={6}>
                                            <Field
                                                name={'http_listen_addr'}
                                                validators={{
                                                    onChange: z.string()
                                                }}
                                                children={(props) => {
                                                    return (
                                                        <TextFieldSimple
                                                            {...props}
                                                            label={t(
                                                                'settings.http.http_listen_addr_label'
                                                            )}
                                                            tooltip={t(
                                                                'settings.http.http_listen_addr_tooltip'
                                                            )}
                                                        />
                                                    );
                                                }}
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
                                slotProps={{
                                    transition: { unmountOnExit: true }
                                }}
                                expanded={expanded === 'steam'}
                                onChange={handleChange('steam')}
                            >
                                <AccordionSummary
                                    expandIcon={<ExpandMoreIcon />}
                                    aria-controls="steam-content"
                                    id="steam-header"
                                >
                                    <Typography
                                        sx={{ width: '33%', flexShrink: 0 }}
                                    >
                                        <Trans
                                            i18nKey={'settings.steam.label'}
                                        />
                                    </Typography>
                                    <Typography
                                        sx={{ color: 'text.secondary' }}
                                    >
                                        <Trans
                                            i18nKey={
                                                'settings.steam.description'
                                            }
                                        />
                                    </Typography>
                                </AccordionSummary>
                                <AccordionDetails>
                                    <Grid container>
                                        <Grid xs={6}>
                                            <Field
                                                name={'steam_id'}
                                                validators={{
                                                    onChange: z.string()
                                                }}
                                                children={(props) => {
                                                    return (
                                                        <TextFieldSimple
                                                            {...props}
                                                            label={t(
                                                                'settings.steam.steam_id_label'
                                                            )}
                                                            tooltip={t(
                                                                'settings.steam.steam_id_tooltip'
                                                            )}
                                                        />
                                                    );
                                                }}
                                            />
                                        </Grid>

                                        <Grid xs={6}>
                                            <Field
                                                name={'api_key'}
                                                validators={{
                                                    onChange: z
                                                        .string()
                                                        .length(32)
                                                }}
                                                children={(props) => {
                                                    return (
                                                        <TextFieldSimple
                                                            {...props}
                                                            label={t(
                                                                'settings.steam.api_key_label'
                                                            )}
                                                            tooltip={t(
                                                                'settings.steam.api_key_tooltip'
                                                            )}
                                                        />
                                                    );
                                                }}
                                            />
                                        </Grid>
                                        <Grid xs={12}>
                                            <Field
                                                name={'steam_dir'}
                                                validators={{
                                                    onChange: z.string()
                                                }}
                                                children={(props) => {
                                                    return (
                                                        <TextFieldSimple
                                                            {...props}
                                                            label={t(
                                                                'settings.steam.steam_dir_label'
                                                            )}
                                                            tooltip={t(
                                                                'settings.steam.steam_dir_tooltip'
                                                            )}
                                                        />
                                                    );
                                                }}
                                            />
                                        </Grid>
                                        <Grid xs={6}>
                                            <Field
                                                name={'bd_api_enabled'}
                                                validators={{
                                                    onSubmit: z.boolean()
                                                }}
                                                children={(props) => {
                                                    return (
                                                        <CheckboxSimple
                                                            {...props}
                                                            label={t(
                                                                'settings.steam.bd_api_enabled_label'
                                                            )}
                                                            tooltip={t(
                                                                'settings.steam.bd_api_enabled_tooltip'
                                                            )}
                                                        />
                                                    );
                                                }}
                                            />
                                        </Grid>
                                        <Grid xs={6}>
                                            <Field
                                                name={'bd_api_address'}
                                                validators={{
                                                    onChange: z.string()
                                                }}
                                                children={(props) => {
                                                    return (
                                                        <TextFieldSimple
                                                            {...props}
                                                            label={t(
                                                                'settings.steam.bd_api_address_label'
                                                            )}
                                                            tooltip={t(
                                                                'settings.steam.bd_api_address_tooltip'
                                                            )}
                                                        />
                                                    );
                                                }}
                                            />
                                        </Grid>
                                    </Grid>
                                </AccordionDetails>
                            </Accordion>

                            <Accordion
                                slotProps={{
                                    transition: { unmountOnExit: true }
                                }}
                                expanded={expanded === 'tf2'}
                                onChange={handleChange('tf2')}
                            >
                                <AccordionSummary
                                    expandIcon={<ExpandMoreIcon />}
                                    aria-controls="tf2-content"
                                    id="tf2-header"
                                >
                                    <Typography
                                        sx={{ width: '33%', flexShrink: 0 }}
                                    >
                                        <Trans i18nKey={'settings.tf2.label'} />
                                    </Typography>
                                    <Typography
                                        sx={{ color: 'text.secondary' }}
                                    >
                                        <Trans i18nKey={'settings.tf2.label'} />
                                    </Typography>
                                </AccordionSummary>
                                <AccordionDetails>
                                    <Grid container>
                                        <Grid xs={12}>
                                            <Field
                                                name={'tf2_dir'}
                                                validators={{
                                                    onChange: z.string()
                                                }}
                                                children={(props) => {
                                                    return (
                                                        <TextFieldSimple
                                                            {...props}
                                                            label={t(
                                                                'settings.tf2.tf2_dir_label'
                                                            )}
                                                            tooltip={t(
                                                                'settings.tf2.tf2_dir_tooltip'
                                                            )}
                                                        />
                                                    );
                                                }}
                                            />
                                        </Grid>
                                        <Grid xs={6}>
                                            <Field
                                                name={'rcon_static'}
                                                validators={{
                                                    onSubmit: z.boolean()
                                                }}
                                                children={(props) => {
                                                    return (
                                                        <CheckboxSimple
                                                            {...props}
                                                            label={t(
                                                                'settings.tf2.rcon_static_label'
                                                            )}
                                                            tooltip={t(
                                                                'settings.tf2.rcon_static_tooltip'
                                                            )}
                                                        />
                                                    );
                                                }}
                                            />
                                        </Grid>
                                        <Grid xs={6}>
                                            <Field
                                                name={'voice_bans_enabled'}
                                                validators={{
                                                    onSubmit: z.boolean()
                                                }}
                                                children={(props) => {
                                                    return (
                                                        <CheckboxSimple
                                                            {...props}
                                                            label={t(
                                                                'settings.tf2.voice_bans_enabled_label'
                                                            )}
                                                            tooltip={t(
                                                                'settings.tf2.voice_bans_enabled_tooltip'
                                                            )}
                                                        />
                                                    );
                                                }}
                                            />
                                        </Grid>
                                    </Grid>
                                </AccordionDetails>
                            </Accordion>
                        </Grid>
                        <Grid xs={12} padding={2} paddingTop={4}>
                            <Subscribe
                                selector={(state) => [
                                    state.canSubmit,
                                    state.isSubmitting
                                ]}
                                children={([canSubmit, isSubmitting]) => {
                                    return (
                                        <Buttons
                                            reset={reset}
                                            canSubmit={canSubmit}
                                            isSubmitting={isSubmitting}
                                        />
                                    );
                                }}
                            />
                        </Grid>
                    </Grid>
                </form>
            </Grid>
        </Grid>
    );
}
