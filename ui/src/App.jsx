import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import { ThemeProvider, createTheme, CssBaseline } from '@mui/material';
import LoginPage from './components/login/LoginPage.jsx';
import ChatPage from './components/chat/ChatPage';

const theme = createTheme({
    palette: {
        primary: {
            main: '#E91E63', // Pink
            light: '#F48FB1',
            dark: '#C2185B',
            contrastText: '#fff',
        },
        secondary: {
            main: '#FF4081', // Pink accent
            light: '#FF80AB',
            dark: '#F50057',
            contrastText: '#fff',
        },
        background: {
            default: '#FDF5F7', // Very light pink background
            paper: '#fff',
        },
    },
    typography: {
        fontFamily: '"Roboto", "Helvetica", "Arial", sans-serif',
        h4: {
            fontWeight: 600,
        },
        h6: {
            fontWeight: 500,
        },
    },
    shape: {
        borderRadius: 8,
    },
    components: {
        MuiButton: {
            styleOverrides: {
                root: {
                    textTransform: 'none',
                    fontWeight: 500,
                },
                contained: {
                    boxShadow: 'none',
                    '&:hover': {
                        boxShadow: '0px 2px 4px -1px rgba(0,0,0,0.2)',
                    },
                },
            },
        },
        MuiPaper: {
            styleOverrides: {
                elevation3: {
                    boxShadow: '0px 3px 8px rgba(233, 30, 99, 0.15)',
                },
            },
        },
    },
});

function App() {
    return (
        <ThemeProvider theme={theme}>
            <CssBaseline />
            <Router>
                <Routes>
                    <Route path="/login" element={<LoginPage />} />
                    <Route path="/chat" element={<ChatPage />} />
                    <Route path="/chat" element={<ChatPage />} />
                    <Route path="/" element={<Navigate to="/login" />} />
                </Routes>
            </Router>
        </ThemeProvider>
    );
}

export default App;