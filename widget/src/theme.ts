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
            MuiTableCell: {
                styleOverrides: {
                    root: {
                        border: 0
                    }
                }
            },
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
            fontFamily: baseFontSet.join(','),
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
                main: '#395c78',
                dark: '#2b475d'
            },
            secondary: {
                main: '#9d312f',
                dark: '#d14441'
            },
            background: {
                default: '#0c0c0c',
                paper: '#151515'
            },
            common: {
                white: '#f5e7de',
                black: '#34302d'
            },
            text: {
                primary: '#b6b1af',
                secondary: '#7b7775'
            },
            divider: '#3a3a3a'
        }
    });
};
