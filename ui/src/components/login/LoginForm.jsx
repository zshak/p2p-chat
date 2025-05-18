import React from 'react';
import {
    Box,
    Typography,
    TextField,
    Button,
    Alert,
    Avatar,
    Paper,
} from '@mui/material';
import LockOutlinedIcon from '@mui/icons-material/LockOutlined';
// import ChatIcon from '@mui/icons-material/Chat';
import chatIcon from '../../../public/icon.svg';

const LoginForm = ({
                       password,
                       setPassword,
                       handlePasswordSubmit,
                       error,
                       isMobile
                   }) => {
    return (
        <Paper
            elevation={3}
            sx={{
                padding: isMobile ? 3 : 4,
                width: '100%',
                borderRadius: 2,
            }}
        >
            <Box sx={{
                display: 'flex',
                flexDirection: 'column',
                alignItems: 'center',
                mb: 3
            }}>
                <img
                    src={chatIcon}
                    alt="Chat Icon"
                    style={{
                        width: '56px',
                        height: '56px',
                        marginBottom: '8px'
                    }}
                />
                <Typography component="h1" variant="h4" align="center" gutterBottom color="primary">
                    P2P Chat
                </Typography>
                <Typography variant="body2" color="text.secondary" align="center">
                    Secure peer-to-peer communication
                </Typography>
            </Box>

            {error && <Alert severity="error" sx={{ mb: 2, borderRadius: 1 }}>{error}</Alert>}

            <Box component="form" onSubmit={handlePasswordSubmit} sx={{ mt: 1 }}>
                <Box sx={{
                    display: 'flex',
                    alignItems: 'center',
                    mb: 2
                }}>
                    <Avatar sx={{ bgcolor: 'secondary.main', mr: 2 }}>
                        <LockOutlinedIcon />
                    </Avatar>
                    <Typography variant="h6">
                        Enter Password
                    </Typography>
                </Box>
                <TextField
                    margin="normal"
                    required
                    fullWidth
                    name="password"
                    label="Password"
                    type="password"
                    id="password"
                    autoComplete="current-password"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    autoFocus
                    variant="outlined"
                    sx={{ mb: 2 }}
                />
                <Button
                    type="submit"
                    fullWidth
                    variant="contained"
                    color="primary"
                    onClick={handlePasswordSubmit}
                    size="large"
                    sx={{
                        mt: 1,
                        mb: 2,
                        py: 1.5,
                        borderRadius: 1.5
                    }}
                >
                    Login
                </Button>
            </Box>
        </Paper>
    );
};

export default LoginForm;