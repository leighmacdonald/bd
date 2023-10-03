import React, { Fragment, useMemo } from 'react';
import Container from '@mui/material/Container';
import CssBaseline from '@mui/material/CssBaseline';
import ThemeProvider from '@mui/material/styles/ThemeProvider';
import { BrowserRouter as Router, Route, Routes } from 'react-router-dom';
import { ErrorBoundary } from './component/ErrorBoundary';
import { createThemeByMode } from './theme';
import { Home } from './page/Home';
import { useUserSettings } from './api';
import { SettingsContext } from './context/settings';
import NiceModal from '@ebay/nice-modal-react';
import { NoteEditor } from './component/NoteEditor';
import { SettingsEditor } from './component/SettingsEditor';

export const ModalNotes = 'modal-notes';
export const ModalSettings = 'modal-settings';

NiceModal.register(ModalNotes, NoteEditor);
NiceModal.register(ModalSettings, SettingsEditor);

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
