import * as path from 'path';
import HtmlWebpackPlugin from 'html-webpack-plugin';
import type { Configuration as DevServerConfiguration } from 'webpack-dev-server';
import type { Configuration } from 'webpack';

const outPath = path.resolve('../internal/detector/dist');

const devMode = process.env.NODE_ENV !== 'production';

const paths = {
    src: path.join(__dirname, 'src'),
    dist: outPath
};

const devServer: DevServerConfiguration = {
    static: {
        directory: paths.dist
    },
    hot: true,
    port: 9000
};

const config: Configuration = {
    entry: './src/index.tsx',
    output: {
        path: path.join(paths.dist),
        publicPath: '/dist/',
        filename: devMode ? '[name].js' : '[name].[chunkhash:8].bundle.js',
        clean: false
    },
    devtool: devMode ? 'inline-source-map' : false,
    devServer,
    performance: {
        maxAssetSize: 1000000,
        maxEntrypointSize: 1000000
    },
    optimization: {
        // runtimeChunk: 'single',
        splitChunks: {
            chunks: 'all',
            // chunks: 'async',
            minSize: 2000,
            minRemainingSize: 0,
            minChunks: 10,
            maxAsyncRequests: 3,
            maxInitialRequests: 3,
            enforceSizeThreshold: 5000,
            cacheGroups: {
                defaultVendors: {
                    test: /[\\/]node_modules[\\/]/,
                    priority: -10,
                    reuseExistingChunk: true
                },
                default: {
                    minChunks: 2,
                    priority: -20,
                    reuseExistingChunk: true
                }
            }
        }
    },
    module: {
        rules: [
            {
                test: /\.tsx?$/,
                use: 'ts-loader',
                exclude: /node_modules/
            },
            {
                test: /\.(scss|css)$/,
                use: ['style-loader', 'css-loader', 'sass-loader']
            },
            {
                test: /\.(jpg|png|svg)$/,
                loader: 'url-loader',
                options: {
                    limit: 250000
                }
            },
            {
                test: /\.(woff|otf|ttf|svg|eot)$/,
                type: 'asset/resource',
                generator: {
                    filename: 'static/[hash][ext][query]'
                }
            }
        ]
    },
    resolve: {
        extensions: ['.tsx', '.ts', '.js']
    },
    plugins: [
        new HtmlWebpackPlugin({
            template: path.join(paths.src, 'index.html'),
            filename: path.join(paths.dist, 'index.html'),
            inject: true,
            hash: !devMode,
            minify: {
                removeComments: !devMode,
                collapseWhitespace: !devMode,
                minifyJS: !devMode,
                minifyCSS: !devMode
            }
        })
    ]
};

export default config;
