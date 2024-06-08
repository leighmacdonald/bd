import { createRootRouteWithContext, Outlet } from '@tanstack/react-router';
import CssBaseline from '@mui/material/CssBaseline';
import Container from '@mui/material/Container';
import { ErrorBoundary } from '../component/ErrorBoundary.tsx';
import { QueryClient, useQuery } from '@tanstack/react-query';
import { getStateOptions, getUserSettingsQuery } from '../api.ts';
import { useSettingsState } from '../context/SettingsContext.ts';

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
                    <ErrorBoundary>
                        <Outlet />
                    </ErrorBoundary>
                </Container>
            </>
        );
    }
});
