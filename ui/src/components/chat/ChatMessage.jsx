import React from 'react';
import {
    Box,
    Paper,
    Typography,
    Avatar,
} from '@mui/material';

const ChatMessage = ({ message, currentUser }) => {
    const isMyMessage = message.sender === 'me' || message.sender === currentUser?.peerId;

    const formatTime = (timestamp) => {
        const date = new Date(timestamp);
        return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    };

    const getSenderInitial = () => {
        if (isMyMessage) return 'Y'; // Your initial
        return message.senderName ? message.senderName.charAt(0).toUpperCase() : 'U';
    };

    return (
        <Box
            sx={{
                display: 'flex',
                justifyContent: isMyMessage ? 'flex-end' : 'flex-start',
                mb: 2,
            }}
        >
            {!isMyMessage && (
                <Avatar sx={{ bgcolor: 'secondary.main', mr: 1 }}>
                    {getSenderInitial()}
                </Avatar>
            )}
            <Box
                sx={{
                    maxWidth: '70%',
                    minWidth: '100px',
                }}
            >
                <Paper
                    elevation={1}
                    sx={{
                        p: 1.5,
                        borderRadius: 2,
                        bgcolor: isMyMessage ? 'primary.main' : 'background.paper',
                        color: isMyMessage ? 'primary.contrastText' : 'text.primary',
                    }}
                >
                    <Typography variant="body1">{message.text}</Typography>
                </Paper>
                <Typography
                    variant="caption"
                    color="text.secondary"
                    sx={{
                        display: 'block',
                        mt: 0.5,
                        textAlign: isMyMessage ? 'right' : 'left',
                    }}
                >
                    {formatTime(message.timestamp)}
                </Typography>
            </Box>
            {isMyMessage && (
                <Avatar sx={{ bgcolor: 'primary.dark', ml: 1 }}>
                    Y
                </Avatar>
            )}
        </Box>
    );
};

export default ChatMessage;