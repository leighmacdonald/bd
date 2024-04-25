import React from 'react';
import { useTranslation } from 'react-i18next';
import ButtonGroup from '@mui/material/ButtonGroup';
import Tooltip from '@mui/material/Tooltip';
import Box from '@mui/material/Box';
import IconButton from '@mui/material/IconButton';
import FilterListOutlinedIcon from '@mui/icons-material/FilterListOutlined';
import NiceModal from '@ebay/nice-modal-react';
import Stack from '@mui/material/Stack';
import SettingsOutlinedIcon from '@mui/icons-material/SettingsOutlined';
import PlayArrowIcon from '@mui/icons-material/PlayArrow';
import StopIcon from '@mui/icons-material/Stop';
import Typography from '@mui/material/Typography';
import { logError } from '../util';
import {
    getLaunch,
    getQuit,
    getState,
    getUserSettings,
    saveUserSettings,
    Team,
    UserSettings
} from '../api';
import { ColumnConfigButton } from './PlayerTable';
import { ModalSettings } from './modal';
import { useMutation, useQuery } from '@tanstack/react-query';
import { getMatchesOnly } from '../table.ts';

export const Toolbar = () => {
    const { t } = useTranslation();

    const { data: state } = useQuery({
        queryKey: ['state'],
        queryFn: getState
    });

    const { data: settings } = useQuery({
        queryKey: ['settings'],
        queryFn: getUserSettings
    });

    const { data: matchesOnly } = useQuery({
        queryKey: ['matchesOnly'],
        queryFn: getMatchesOnly
    });

    const matchesOnlyMut = useMutation({
        mutationKey: ['matchesOnly'],
        mutationFn: async (newValue: boolean) => {
            localStorage.setItem(
                'matchesOnly',
                JSON.stringify(Boolean(newValue))
            );
            return matchesOnly;
        }
    });

    const settingsMutation = useMutation({
        mutationFn: saveUserSettings
    });

    return (
        <Stack direction={'row'}>
            <ButtonGroup>
                <Tooltip title={t('toolbar.button.show_only_negative')}>
                    <Box>
                        <IconButton
                            onClick={() => matchesOnlyMut.mutate(!matchesOnly)}
                        >
                            <FilterListOutlinedIcon color={'primary'} />
                        </IconButton>
                    </Box>
                </Tooltip>

                <Tooltip title={t('toolbar.button.shown_columns')}>
                    <Box>
                        <ColumnConfigButton />
                    </Box>
                </Tooltip>

                <Tooltip title={t('toolbar.button.open_settings')}>
                    <Box>
                        <IconButton
                            onClick={async () => {
                                try {
                                    const newSettings =
                                        await NiceModal.show<UserSettings>(
                                            ModalSettings,
                                            { settings }
                                        );
                                    settingsMutation.mutate(newSettings);
                                } catch (e) {
                                    logError(e);
                                }
                            }}
                        >
                            <SettingsOutlinedIcon color={'primary'} />
                        </IconButton>
                    </Box>
                </Tooltip>

                <Tooltip
                    title={t(
                        state
                            ? state.game_running
                                ? 'toolbar.button.game_state_running'
                                : 'toolbar.button.game_state_stopped'
                            : 'toolbar.button.game_state_stopped'
                    )}
                >
                    <Box>
                        <IconButton
                            color={
                                state
                                    ? !state.game_running
                                        ? 'success'
                                        : 'error'
                                    : 'success'
                            }
                            onClick={
                                state
                                    ? !state.game_running
                                        ? getLaunch
                                        : getQuit
                                    : getLaunch
                            }
                            disabled={!state}
                        >
                            {state ? (
                                state.game_running ? (
                                    <StopIcon />
                                ) : (
                                    <PlayArrowIcon />
                                )
                            ) : (
                                <PlayArrowIcon />
                            )}
                        </IconButton>
                    </Box>
                </Tooltip>
            </ButtonGroup>
            <Box sx={{ display: 'flex', alignItems: 'center' }} paddingLeft={1}>
                <Typography variant={'button'} style={{ color: '#004ec2' }}>
                    {state
                        ? (state.players ?? []).filter(
                              (p) => p.team == Team.BLU && p.is_connected
                          ).length
                        : 0}
                </Typography>
                <Typography
                    variant={'button'}
                    style={{ paddingLeft: 3, paddingRight: 3 }}
                >
                    :
                </Typography>
                <Typography variant={'button'} style={{ color: '#b40a2a' }}>
                    {state
                        ? state.players.filter(
                              (p) => p.team == Team.RED && p.is_connected
                          ).length
                        : 0}
                </Typography>
            </Box>
            <Box sx={{ display: 'flex', alignItems: 'center' }} paddingLeft={2}>
                <Typography variant={'h1'}>
                    {state ? state.server.server_name : 'Not Connected'}
                </Typography>
            </Box>
            <Box sx={{ display: 'flex', alignItems: 'center' }} paddingLeft={2}>
                <Typography variant={'subtitle1'} paddingRight={1}>
                    {state ? state.server.current_map : 'N/A'}
                </Typography>
            </Box>
        </Stack>
    );
};
