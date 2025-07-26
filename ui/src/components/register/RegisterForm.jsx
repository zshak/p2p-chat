import React, {useState} from 'react';
import {Alert, Avatar, Box, Button, Paper, TextField, Typography,} from '@mui/material';
import LockOutlinedIcon from '@mui/icons-material/LockOutlined';
import chatIcon from '../../../public/icon.svg';

const RegisterForm = ({handleRegister, error, loading, isMobile}) => {
    const [password, setPassword] = useState('');
    const [confirmPassword, setConfirmPassword] = useState('');
    const [localError, setLocalError] = useState('');

    const handleSubmit = (e) => {
        e.preventDefault();
        setLocalError('');
        if (!password.trim()) {
            setLocalError('Password cannot be empty');
            return;
        }
        if (password !== confirmPassword) {
            setLocalError('Passwords do not match');
            return;
        }
        if (password.length < 6) {
            setLocalError('Password must be at least 6 characters long');
            return;
        }
        handleRegister(password);
    };

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

            {(error || localError) && (
                <Alert severity="error" sx={{mb: 2, borderRadius: 1}}>
                    {error || localError}
                </Alert>
            )}

            <Box component="form" onSubmit={handleSubmit} sx={{mt: 1}}>
                <Box sx={{
                    display: 'flex',
                    alignItems: 'center',
                    mb: 2
                }}>
                    <Avatar sx={{bgcolor: 'secondary.main', mr: 2}}>
                        <LockOutlinedIcon/>
                    </Avatar>
                    <Typography variant="h6">
                        Create Account
                    </Typography>
                </Box>

                <Typography variant="body2" color="text.secondary" sx={{mb: 2}}>
                    Set up your secure chat by creating a password to protect your encryption keys.
                </Typography>

                <TextField
                    margin="normal"
                    required
                    fullWidth
                    name="password"
                    label="Password"
                    type="password"
                    id="password"
                    autoComplete="new-password"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    variant="outlined"
                    sx={{mb: 2}}
                    disabled={loading}
                    autoFocus
                />
                <TextField
                    margin="normal"
                    required
                    fullWidth
                    name="confirmPassword"
                    label="Confirm Password"
                    type="password"
                    id="confirmPassword"
                    autoComplete="new-password"
                    value={confirmPassword}
                    onChange={(e) => setConfirmPassword(e.target.value)}
                    variant="outlined"
                    sx={{mb: 2}}
                    disabled={loading}
                />
                <Button
                    type="submit"
                    fullWidth
                    variant="contained"
                    color="primary"
                    size="large"
                    disabled={loading}
                    sx={{
                        mt: 1,
                        mb: 2,
                        py: 1.5,
                        borderRadius: 1.5
                    }}
                >
                    {loading ? 'Creating Account...' : 'Register & Create Account'}
                </Button>
            </Box>
        </Paper>
    );
};

export default RegisterForm;