import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react-swc';
import { createHtmlPlugin } from 'vite-plugin-html';

export default defineConfig({
    //base: '/',
    plugins: [
        react(),
        createHtmlPlugin({
            entry: 'src/index.tsx',
            template: 'index.html',
            inject: {
                data: {
                    title: 'bd',
                    version: 'v0.0.1'
                }
            }
        })
    ]
});
