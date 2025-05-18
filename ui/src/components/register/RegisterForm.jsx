import React from 'react';
import {
    Box,
    Typography,
    Button,
    Paper,
} from '@mui/material';

const RegisterForm = ({ handleRegister }) => {
    return (
        <Box sx={{ mt: 2 }}>
            <Typography variant="h6" gutterBottom align="center" color="primary.dark">
                Welcome to P2P Chat
            </Typography>
            <Typography variant="body1" paragraph align="center" sx={{ mb: 3 }}>
                It looks like you're new here. Set up your secure chat by registering below.
            </Typography>
            <Button
                fullWidth
                variant="contained"
                color="primary"
                onClick={handleRegister}
                size="large"
                sx={{
                    mt: 2,
                    py: 1.5,
                    borderRadius: 1.5
                }}
            >
                Register & Create Account
            </Button>
        </Box>
    );
};

export default RegisterForm;