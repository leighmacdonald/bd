import createTheme from '@mui/material/styles/createTheme';
import darkScrollbar from '@mui/material/darkScrollbar';

const baseFontSet = [
    '"Helvetica Neue"',
    'Helvetica',
    'Roboto',
    'Arial',
    'sans-serif'
];

export const createThemeByMode = () => {

    return createTheme({
        components: {
            MuiCssBaseline: {
                styleOverrides: {
                    body: darkScrollbar()
                }
            },
            MuiButton: {
                variants: [
                    {
                        props: { variant: 'contained' },
                        style: baseFontSet
                    }
                ]
            }
        },
        typography: {
            fontFamily: [
                '"Helvetica Neue"',
                'Helvetica',
                'Roboto',
                'Arial',
                'sans-serif'
            ].join(','),
            // allVariants: {
            //     color: mode === 'dark' ? '#34302d' : '#f5e7de'
            // },
            body1: {
                //...baseFontSet
            },
            fontSize: 12,
            h1: {
                fontSize: 36
            },
            h2: {
                fontSize: 32
            },
            h3: {
                fontSize: 28
            },
            h4: {
                fontSize: 24
            },
            h5: {
                fontSize: 20
            },
            h6: {
                fontSize: 16
            }
        },
        palette: {
            primary: {
                main: '#9d312f',
                dark: '#d14441'
            },
            secondary: {
                main: '#395c78'
            },
            background: {
                default: '#0c0c0c',
                paper: '#0c0c0c'
            },
            common: {
                white: '#f5e7de',
                black: '#34302d'
            },
            text: {
                primary: '#f5e7de',
                secondary: '#e3d6ce'
            },
            divider: '#452c22'
        }
    });
};
