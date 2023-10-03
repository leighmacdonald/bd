import React, { Fragment, useMemo } from 'react';
import { BrowserRouter as Router, Route, Routes } from 'react-router-dom';
import CssBaseline from '@mui/material/CssBaseline';
import Container from '@mui/material/Container';
import ThemeProvider from '@mui/material/styles/ThemeProvider';
import NiceModal from '@ebay/nice-modal-react';
import { ErrorBoundary } from './component/ErrorBoundary';
import { createThemeByMode } from './theme';
import { Home } from './page/Home';
import { useUserSettings } from './api';
import { SettingsContext } from './context/SettingsContext';
import './modals';

export const App = (): JSX.Element => {
    const theme = useMemo(() => createThemeByMode(), []);
    const userSettings = useUserSettings();

    return (
        <Router>
            <React.Fragment>
                <ThemeProvider theme={theme}>
                    <React.StrictMode>
                        <NiceModal.Provider>
                            <CssBaseline />
                            <Container maxWidth={'lg'} disableGutters>
                                {!userSettings.loading && (
                                    <SettingsContext.Provider
                                        value={userSettings}
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
                    </React.StrictMode>
                </ThemeProvider>
            </React.Fragment>
        </Router>
    );
};
