import React from 'react';
import Grid2 from '@mui/material/Unstable_Grid2';
import { PlayerTable } from '../component/PlayerTable';

export const Home = () => {
    return (
        <Grid2 container>
            <Grid2 xs={12}>
                <PlayerTable />
            </Grid2>
        </Grid2>
    );
};
