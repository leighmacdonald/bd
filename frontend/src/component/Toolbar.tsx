import { useCallback, useContext, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import ButtonGroup from '@mui/material/ButtonGroup';
import Tooltip from '@mui/material/Tooltip';
import Box from '@mui/material/Box';
import IconButton from '@mui/material/IconButton';
import FilterListOutlinedIcon from '@mui/icons-material/FilterListOutlined';
import Stack from '@mui/material/Stack';
import SettingsOutlinedIcon from '@mui/icons-material/SettingsOutlined';
import PlayArrowIcon from '@mui/icons-material/PlayArrow';
import StopIcon from '@mui/icons-material/Stop';
import Typography from '@mui/material/Typography';
import { getLaunchOptions, getQuitOptions, Team } from '../api';
import { ColumnConfigButton } from './PlayerTable';
import { PlayerTableContext } from '../context/PlayerTableContext';
import { useQueryClient } from '@tanstack/react-query';
import { useGameState } from '../context/GameStateContext.ts';
import { useNavigate } from '@tanstack/react-router';
import HomeIcon from '@mui/icons-material/Home';

export const Toolbar = () => {
    const { state } = useGameState();
    const { t } = useTranslation();
    const { setMatchesOnly } = useContext(PlayerTableContext);
    const queryClient = useQueryClient();
    const navigate = useNavigate();
    const isSettings = window.location.pathname == '/settings';

    const onSetMatches = useCallback(() => {
        setMatchesOnly((prevState) => {
            localStorage.setItem('matchesOnly', `${!prevState}`);
            return !prevState;
        });
    }, [setMatchesOnly]);

    const gameRunningQuery = useMemo(() => {
        if (!state.game_running) {
            return getLaunchOptions();
        } else {
            return getQuitOptions();
        }
    }, [state.game_running]);

    return (
        <Stack direction={'row'}>
            <ButtonGroup>
                <Tooltip title={t('toolbar.button.show_only_negative')}>
                    <Box>
                        <IconButton onClick={onSetMatches}>
                            <FilterListOutlinedIcon color={'primary'} />
                        </IconButton>
                    </Box>
                </Tooltip>

                <Tooltip title={t('toolbar.button.shown_columns')}>
                    <Box>
                        <ColumnConfigButton />
                    </Box>
                </Tooltip>
                {!isSettings ? (
                    <Tooltip title={t('toolbar.button.open_settings')}>
                        <Box>
                            <IconButton
                                onClick={async () => {
                                    await navigate({ to: '/settings' });
                                }}
                            >
                                <SettingsOutlinedIcon color={'primary'} />
                            </IconButton>
                        </Box>
                    </Tooltip>
                ) : (
                    <Tooltip title={t('toolbar.button.open_home')}>
                        <Box>
                            <IconButton
                                onClick={async () => {
                                    await navigate({ to: '/' });
                                }}
                            >
                                <HomeIcon color={'primary'} />
                            </IconButton>
                        </Box>
                    </Tooltip>
                )}
                <Tooltip
                    title={t(
                        state.game_running
                            ? 'toolbar.button.game_state_running'
                            : 'toolbar.button.game_state_stopped'
                    )}
                >
                    <Box>
                        <IconButton
                            color={!state.game_running ? 'success' : 'error'}
                            onClick={async () => {
                                await queryClient.fetchQuery(gameRunningQuery);
                            }}
                        >
                            {!state.game_running ? (
                                <PlayArrowIcon />
                            ) : (
                                <StopIcon />
                            )}
                        </IconButton>
                    </Box>
                </Tooltip>
            </ButtonGroup>
            <Box sx={{ display: 'flex', alignItems: 'center' }} paddingLeft={1}>
                <Typography variant={'button'} style={{ color: '#004ec2' }}>
                    {
                        state.players.filter(
                            (p) => p.team == Team.BLU && p.is_connected
                        ).length
                    }
                </Typography>
                <Typography
                    variant={'button'}
                    style={{ paddingLeft: 3, paddingRight: 3 }}
                >
                    :
                </Typography>
                <Typography variant={'button'} style={{ color: '#b40a2a' }}>
                    {
                        state.players.filter(
                            (p) => p.team == Team.RED && p.is_connected
                        ).length
                    }
                </Typography>
            </Box>
            ;
            <Box sx={{ display: 'flex', alignItems: 'center' }} paddingLeft={2}>
                <Typography variant={'h1'}>
                    {state.server.server_name}
                </Typography>
            </Box>
            ;
            <Box sx={{ display: 'flex', alignItems: 'center' }} paddingLeft={2}>
                <Typography variant={'subtitle1'} paddingRight={1}>
                    {state.server.current_map}
                </Typography>
            </Box>
            ;
        </Stack>
    );
};
