import { createRootRoute, Outlet } from '@tanstack/react-router';
import CssBaseline from '@mui/material/CssBaseline';
import Container from '@mui/material/Container';
import { ErrorBoundary } from '../component/ErrorBoundary.tsx';

export const Route = createRootRoute({
    component: () => {
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
