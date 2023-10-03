import React, { useContext } from 'react';
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
import { ModalSettings } from '../modals';
import { logError } from '../util';
import { Team, useCurrentState } from '../api';
import { ColumnConfigButton } from './PlayerTable';
import { PlayerTableContext } from '../context/PlayerTableContext';

export const Toolbar = () => {
    const state = useCurrentState();
    const { t } = useTranslation();
    const { setMatchesOnly } = useContext(PlayerTableContext);
    return (
        <Stack direction={'row'}>
            <ButtonGroup>
                <Tooltip title={t('toolbar.button.show_only_negative')}>
                    <Box>
                        <IconButton
                            onClick={() => {
                                setMatchesOnly((prevState) => {
                                    localStorage.setItem(
                                        'matchesOnly',
                                        `${!prevState}`
                                    );
                                    return !prevState;
                                });
                            }}
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
                                    await NiceModal.show(ModalSettings);
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
                            disableRipple
                            color={state.game_running ? 'success' : 'error'}
                            onClick={() => {}}
                        >
                            {state.game_running ? (
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
