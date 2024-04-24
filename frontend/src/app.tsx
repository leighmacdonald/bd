import React, { StrictMode } from 'react';
import ReactDOM from 'react-dom/client';
import { RouterProvider, createRouter } from '@tanstack/react-router';

import './component/modal';
import { routeTree } from './routeTree.gen';
import './i18n';

const router = createRouter({ routeTree, defaultPreload: 'intent' });

// Register the router instance for type safety
declare module '@tanstack/react-router' {
    interface Register {
        router: typeof router;
    }
}

// extend window with our own items that we inject
declare global {
    interface Window {
        bd: {
            port: number;
        };
    }
}
const rootElement = document.getElementById('app')!;
if (!rootElement.innerHTML) {
    const root = ReactDOM.createRoot(rootElement);
    root.render(
        <StrictMode>
            <RouterProvider router={router} />
        </StrictMode>
    );
}
