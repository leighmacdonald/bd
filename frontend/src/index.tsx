import React from 'react';
import '@fontsource/roboto/300.css';
import '@fontsource/roboto/400.css';
import '@fontsource/roboto/500.css';
import '@fontsource/roboto/700.css';

import { App } from './App';
import { createRoot } from 'react-dom/client';

import './i18n';

// extend window with our own items that we inject
declare global {
    interface Window {
        bd: {
            port: number;
        };
    }
}

window.bd = window.bd || [];

const container = document.getElementById('root');
if (container) {
    createRoot(container).render(<App />);
}
