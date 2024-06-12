import { createRootRouteWithContext, Outlet } from '@tanstack/react-router';
import CssBaseline from '@mui/material/CssBaseline';
import Container from '@mui/material/Container';
import { ErrorBoundary } from '../component/ErrorBoundary.tsx';
import { QueryClient, useQuery } from '@tanstack/react-query';
import { getStateOptions, getUserSettingsQuery } from '../api.ts';
import { useSettingsState } from '../context/SettingsContext.ts';
import Grid from '@mui/material/Unstable_Grid2';
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import { Toolbar } from '../component/Toolbar.tsx';
import { useEffect } from 'react';
import { useGameState } from '../context/GameStateContext.ts';

interface RouterContext {
    queryClient: QueryClient;
}

export const Route = createRootRouteWithContext<RouterContext>()({
    loader: ({ context }) => {
        return context.queryClient.fetchQuery(getUserSettingsQuery());
    },
    component: () => {
        const settings = Route.useLoaderData();
        const { setSettings } = useSettingsState();
        const { setState } = useGameState();

        useEffect(() => {
            setSettings(settings);
        }, [settings]);

        const { data: state, isLoading } = useQuery({ ...getStateOptions() });

        useEffect(() => {
            if (!isLoading && state) {
                setState(state);
            }
        }, [state]);

        return (
            <>
                <CssBaseline />
                <Container maxWidth={'lg'} disableGutters>
                    <Grid container>
                        <Grid xs={12}>
                            <Paper sx={{ width: '100%', overflow: 'hidden' }}>
                                <Stack>
                                    <Toolbar />
                                    <ErrorBoundary>
                                        <Outlet />
                                    </ErrorBoundary>
                                </Stack>
                            </Paper>
                        </Grid>
                    </Grid>
                </Container>
            </>
        );
    }
});
