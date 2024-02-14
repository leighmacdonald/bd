import { useEffect, useMemo, useState, Fragment, StrictMode } from 'react';
import { BrowserRouter as Router, Route, Routes } from 'react-router-dom';
import CssBaseline from '@mui/material/CssBaseline';
import Container from '@mui/material/Container';
import ThemeProvider from '@mui/material/styles/ThemeProvider';
import NiceModal from '@ebay/nice-modal-react';
import { ErrorBoundary } from './component/ErrorBoundary';
import { createThemeByMode } from './theme';
import { getUserSettings, UserSettings } from './api';
import './component/modal';
import {
    defaultUserSettings,
    SettingsContext
} from './context/SettingsContext';
import { logError } from './util';
import loadable from '@loadable/component';

const Home = loadable(() => import('./page/Home'));

export const App = (): JSX.Element => {
    const theme = useMemo(() => createThemeByMode(), []);
    const [loading, setLoading] = useState(false);
    const [settings, setSettings] = useState<UserSettings>(defaultUserSettings);

    useEffect(() => {
        try {
            setLoading(true);
            getUserSettings().then((newSettings) => {
                setSettings(newSettings);
            });
        } catch (e) {
            logError(e);
        } finally {
            setLoading(false);
        }
    }, []);

    return (
        <Router>
            <Fragment>
                <ThemeProvider theme={theme}>
                    <StrictMode>
                        <NiceModal.Provider>
                            <CssBaseline />
                            <Container maxWidth={'lg'} disableGutters>
                                {!loading && (
                                    <SettingsContext.Provider
                                        value={{ settings, setSettings }}
                                    >
                                        <ErrorBoundary>
                                            <Routes>
                                                <Route
                                                    path={'/'}
                                                    element={
                                                        <ErrorBoundary>
                                                            <Home />
                                                        </ErrorBoundary>
                                                    }
                                                />
                                                <Route
                                                    path="/404"
                                                    element={
                                                        <ErrorBoundary>
                                                            <Fragment>
                                                                not found
                                                            </Fragment>
                                                        </ErrorBoundary>
                                                    }
                                                />
                                            </Routes>
                                        </ErrorBoundary>
                                    </SettingsContext.Provider>
                                )}
                            </Container>
                        </NiceModal.Provider>
                    </StrictMode>
                </ThemeProvider>
            </Fragment>
        </Router>
    );
};
