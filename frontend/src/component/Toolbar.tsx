import { useCallback, useContext } from 'react';
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
    Team,
    useCurrentState,
    UserSettings
} from '../api';
import { ColumnConfigButton } from './PlayerTable';
import { PlayerTableContext } from '../context/PlayerTableContext';
import { ModalSettings } from './modal';
import { SettingsContext } from '../context/SettingsContext.ts';

export const Toolbar = () => {
    const state = useCurrentState();
    const { t } = useTranslation();
    const { setMatchesOnly } = useContext(PlayerTableContext);
    const { settings, setSettings } = useContext(SettingsContext);

    const onSetMatches = useCallback(() => {
        setMatchesOnly((prevState) => {
            localStorage.setItem('matchesOnly', `${!prevState}`);
            return !prevState;
        });
    }, [setMatchesOnly]);

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
                                    setSettings(newSettings);
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
                        state.game_running
                            ? 'toolbar.button.game_state_running'
                            : 'toolbar.button.game_state_stopped'
                    )}
                >
                    <Box>
                        <IconButton
                            color={!state.game_running ? 'success' : 'error'}
                            onClick={!state.game_running ? getLaunch : getQuit}
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
            <Box sx={{ display: 'flex', alignItems: 'center' }} paddingLeft={2}>
                <Typography variant={'h1'}>
                    {state.server.server_name}
                </Typography>
            </Box>
            <Box sx={{ display: 'flex', alignItems: 'center' }} paddingLeft={2}>
                <Typography variant={'subtitle1'} paddingRight={1}>
                    {state.server.current_map}
                </Typography>
            </Box>
        </Stack>
    );
};
