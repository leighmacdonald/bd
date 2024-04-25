import React from 'react';
import Grid from '@mui/material/Unstable_Grid2';
import { PlayerTable } from '../component/PlayerTable.tsx';
import Paper from '@mui/material/Paper';

import Stack from '@mui/material/Stack';
import { Toolbar } from '../component/Toolbar.tsx';

import { createLazyFileRoute } from '@tanstack/react-router';

export const Route = createLazyFileRoute('/')({
    component: Index
});

function Index() {
    return (
        <Grid container>
            <Grid xs={12}>
                <Paper sx={{ width: '100%', overflow: 'hidden' }}>
                    <Stack>
                        <Toolbar />
                        <PlayerTable />
                    </Stack>
                </Paper>
            </Grid>
        </Grid>
    );
}
