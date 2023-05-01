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
                fontSize: 18
            },
            h2: {
                fontSize: 16
            },
            h3: {
                fontSize: 14
            },
            h4: {
                fontSize: 12
            },
            h5: {
                fontSize: 11
            },
            h6: {
                fontSize: 10
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
                primary: '#b6b1af',
                secondary: '#7b7775'
            },
            divider: '#452c22'
        }
    });
};
