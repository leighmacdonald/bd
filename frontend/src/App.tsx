import './component/modal';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { createRouter, RouterProvider } from '@tanstack/react-router';
import { routeTree } from './routeTree.gen.ts';
import { StrictMode, useMemo, useState } from 'react';
import { createThemeByMode } from './theme.ts';
import { GameState, UserSettings } from './api.ts';
import {
    SettingsContext,
    defaultUserSettings
} from './context/SettingsContext.ts';
import { ThemeProvider } from '@mui/material';
import NiceModal from '@ebay/nice-modal-react';
import { GameStateContext } from './context/GameStateContext.ts';

const queryClient = new QueryClient();

const router = createRouter({
    routeTree,
    defaultPreload: 'intent',
    context: {
        queryClient
    },
    // Since we're using React Query, we don't want loader calls to ever be stale
    // This will ensure that the loader is always called when the route is preloaded or visited
    defaultPreloadStaleTime: 0
});

declare module '@tanstack/react-router' {
    // noinspection JSUnusedGlobalSymbols
    interface Register {
        router: typeof router;
    }
}

export const App = (): JSX.Element => {
    const theme = useMemo(() => createThemeByMode(), []);
    const [settings, setSettings] = useState<UserSettings>(defaultUserSettings);
    const [state, setState] = useState<GameState>({
        game_running: false,
        server: {
            server_name: '',
            current_map: '',
            tags: [],
            last_update: ''
        },
        players: []
    });

    return (
        <ThemeProvider theme={theme}>
            <StrictMode>
                <QueryClientProvider client={queryClient}>
                    <SettingsContext.Provider value={{ settings, setSettings }}>
                        <GameStateContext.Provider value={{ setState, state }}>
                            <NiceModal.Provider>
                                <RouterProvider
                                    defaultPreload={'intent'}
                                    router={router}
                                    context={{}}
                                />
                            </NiceModal.Provider>
                        </GameStateContext.Provider>
                    </SettingsContext.Provider>
                </QueryClientProvider>
            </StrictMode>
        </ThemeProvider>
    );
};
