import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react-swc';
import { createHtmlPlugin } from 'vite-plugin-html';

export default defineConfig({
    //base: '/',
    server: {
        open: true,
        port: 8901,
        cors: true,
        proxy: {
            '/api': {
                target: 'http://localhost:8900',
                changeOrigin: true,
                secure: false
            }
        }
    },
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
