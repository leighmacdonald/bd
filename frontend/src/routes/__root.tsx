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

type RouterContext = {
    queryClient: QueryClient;
};

export const Route = createRootRouteWithContext<RouterContext>()({
    loader: ({ context }) => {
        return context.queryClient.fetchQuery(getUserSettingsQuery());
    },
    component: () => {
        const settings = Route.useLoaderData();
        const { setSettings } = useSettingsState();

        setSettings(settings);

        useQuery({ ...getStateOptions() });

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
