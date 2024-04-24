import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import React, { useMemo } from 'react';
import ThemeProvider from '@mui/material/styles/ThemeProvider';
import NiceModal from '@ebay/nice-modal-react';
import CssBaseline from '@mui/material/CssBaseline';
import Container from '@mui/material/Container';
import { ErrorBoundary } from '../component/ErrorBoundary.tsx';
import { ReactQueryDevtools } from '@tanstack/react-query-devtools';
import { createRootRoute, Outlet } from '@tanstack/react-router';
import { TanStackRouterDevtools } from '@tanstack/router-devtools';
import { createThemeByMode } from '../theme.ts';

const queryClient = new QueryClient();

export const Route = createRootRoute({
    component: RootComponent,
    notFoundComponent: () => {
        return <p>Not Found (on root route)</p>;
    }
});

function RootComponent() {
    const theme = useMemo(() => createThemeByMode(), []);

    return (
        <QueryClientProvider client={queryClient}>
            <ThemeProvider theme={theme}>
                <NiceModal.Provider>
                    <CssBaseline />
                    <Container maxWidth={'lg'} disableGutters>
                        <ErrorBoundary>
                            <Outlet />
                        </ErrorBoundary>
                    </Container>
                </NiceModal.Provider>
            </ThemeProvider>

            <ReactQueryDevtools initialIsOpen={false} />
            <TanStackRouterDevtools />
        </QueryClientProvider>
    );
}
