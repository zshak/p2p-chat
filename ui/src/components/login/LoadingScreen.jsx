import React from 'react';
import {
    Box,
    Paper,
    CircularProgress,
    Typography,
    Avatar,
    Container,
    useTheme
} from '@mui/material';
// import ChatIcon from '@mui/icons-material/Chat';
import chatIcon from '../../../public/icon.svg';


const LoadingScreen = ({ loginStep, message, subMessage }) => {
    const theme = useTheme();

    return (
        <Container maxWidth="sm" sx={{
            display: 'flex',
            flexDirection: 'column',
            justifyContent: 'center',
            alignItems: 'center',
            height: '100vh',
            background: theme.palette.background.default
        }}>
            <Paper
                elevation={3}
                sx={{
                    padding: 4,
                    display: 'flex',
                    flexDirection: 'column',
                    alignItems: 'center',
                    borderRadius: 2,
                    width: '100%',
                    maxWidth: 450,
                    minHeight: 350,
                    animation: 'fadeIn 0.3s ease-in-out',
                    '@keyframes fadeIn': {
                        from: {
                            opacity: 0,
                            transform: 'translateY(10px)'
                        },
                        to: {
                            opacity: 1,
                            transform: 'translateY(0)'
                        },
                    },
                }}
            >
                {/*<Avatar sx={{*/}
                {/*    bgcolor: 'primary.main',*/}
                {/*    width: 56,*/}
                {/*    height: 56,*/}
                {/*    mb: 2*/}
                {/*}}>*/}
                {/*    <ChatIcon fontSize="large" />*/}
                {/*</Avatar>*/}

                <img
                    src={chatIcon}
                    alt="Chat Icon"
                    style={{
                        width: '56px',
                        height: '56px',
                        marginBottom: '8px'
                    }}
                />

                <Box sx={{
                    display: 'flex',
                    width: '100%',
                    justifyContent: 'space-between',
                    mb: 3,
                    position: 'relative',
                    px: 2,
                }}>
                    <Box sx={{
                        position: 'absolute',
                        height: '2px',
                        bgcolor: 'secondary.light',
                        width: 'calc(100% - 48px)',
                        left: '24px',
                        top: '12px',
                        zIndex: 0
                    }}/>

                    {[1, 2, 3, 4].map((step) => (
                        <Box key={step} sx={{
                            width: 24,
                            height: 24,
                            borderRadius: '50%',
                            bgcolor: loginStep >= step ? 'primary.main' : 'background.paper',
                            border: '2px solid',
                            borderColor: loginStep >= step ? 'primary.main' : 'secondary.light',
                            zIndex: 1,
                            transition: 'all 0.3s ease',
                            boxShadow: loginStep === step ? '0 0 0 4px rgba(233, 30, 99, 0.3)' : 'none'
                        }}/>
                    ))}
                </Box>

                <CircularProgress color="primary" size={48} thickness={4} sx={{ mb: 2 }} />

                <Typography variant="h6" color="primary" align="center" gutterBottom>
                    {message}
                </Typography>

                <Typography variant="body2" color="text.secondary" align="center">
                    {subMessage}
                </Typography>
            </Paper>
        </Container>
    );
};

export default LoadingScreen;