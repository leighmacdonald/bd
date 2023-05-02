import { PlayerTable } from '../component/PlayerTable';
import React, { useEffect, useState } from 'react';
import Grid2 from '@mui/material/Unstable_Grid2';
import { Box, ButtonGroup, IconButton, Stack, Tooltip } from '@mui/material';
import SettingsOutlinedIcon from '@mui/icons-material/SettingsOutlined';
import FilterListOutlinedIcon from '@mui/icons-material/FilterListOutlined';
import Typography from '@mui/material/Typography';
import { getPlayers, Player } from '../api';

export const Home = () => {
    const [players, setPlayers] = useState<Player[]>([]);
    const [matchesOnly, setMatchesOnly] = useState(
        // Surely strings are the only types
        JSON.parse(localStorage.getItem('matchesOnly') || 'false') === true
    );

    useEffect(() => {
        const interval = setInterval(async () => {
            try {
                setPlayers(await getPlayers());
            } catch (e) {
                console.log(e);
            }
        }, 1000);
        return () => {
            clearInterval(interval);
        };
    }, []);

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
                                <FilterListOutlinedIcon color={'primary'} />
                            </IconButton>
                        </Tooltip>
                        <IconButton>
                            <SettingsOutlinedIcon color={'primary'} />
                        </IconButton>
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
