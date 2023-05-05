import { ColumnConfigButton, PlayerTable } from '../component/PlayerTable';
import React, { useState } from 'react';
import Grid2 from '@mui/material/Unstable_Grid2';
import { Box, ButtonGroup, IconButton, Stack, Tooltip } from '@mui/material';
import SettingsOutlinedIcon from '@mui/icons-material/SettingsOutlined';
import Typography from '@mui/material/Typography';
import { usePlayers } from '../api';

export const Home = () => {
    const players = usePlayers();
    const [matchesOnly, setMatchesOnly] = useState(
        // Surely strings are the only types
        JSON.parse(localStorage.getItem('matchesOnly') || 'false') === true
    );

    return (
        <Grid2 container>
            <Grid2 xs={12}>
                <Stack direction={'row'}>
                    <ButtonGroup>
                        <Tooltip
                            title={
                                'Show only players with some sort of negative status'
                            }
                        >
                            <IconButton
                                onClick={() => {
                                    setMatchesOnly((prevState) => {
                                        console.log(!prevState);
                                        localStorage.setItem(
                                            'matchesOnly',
                                            `${!prevState}`
                                        );
                                        return !prevState;
                                    });
                                }}
                            >
                                <SettingsOutlinedIcon color={'primary'} />
                            </IconButton>
                        </Tooltip>
                        <ColumnConfigButton />
                    </ButtonGroup>
                    <Box sx={{ display: 'flex', alignItems: 'center' }}>
                        <Typography variant={'overline'}></Typography>
                    </Box>
                </Stack>
            </Grid2>
            <Grid2 xs={12}>
                <PlayerTable matchesOnly={matchesOnly} players={players} />
            </Grid2>
        </Grid2>
    );
};
