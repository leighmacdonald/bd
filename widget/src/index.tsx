import React from 'react';
import '@fontsource/roboto/300.css';
import '@fontsource/roboto/400.css';
import '@fontsource/roboto/500.css';
import '@fontsource/roboto/700.css';

import { App } from './App';
import { createRoot } from 'react-dom/client';

// extend window with our own items that we inject
declare global {
    interface Window {
        gbans: {
            siteName: string;
            discordClientId: string;
            discordLinkId: string;
        };
    }
}

window.gbans = window.gbans || [];

const container = document.getElementById('root');
if (container) {
    createRoot(container).render(<App />);
}
