import React from 'react';
import {
    Box,
    Paper,
    Typography,
    Avatar,
} from '@mui/material';

const ChatMessage = ({ message, currentUser }) => {
    // Get current user's peer ID from props or localStorage
    const ownPeerId = currentUser?.peerId || localStorage.getItem('userPeerId') || 'mock-peer-id';
    const isMyMessage = message.sender === 'me' || message.sender === ownPeerId;

    const formatTime = (timestamp) => {
        if (!timestamp) return '';
        const date = new Date(timestamp);
        if (isNaN(date.getTime())) {
            console.error("Invalid timestamp:", timestamp);
            return '';
        }
        return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    };

    const getSenderDisplayName = (senderPeerId) => {
        if (senderPeerId === ownPeerId) return 'You';

        // Fallback to truncated peer ID
        if (senderPeerId && senderPeerId.length > 8) {
            return `${senderPeerId.substring(0, 4)}...${senderPeerId.substring(senderPeerId.length - 4)}`;
        }
        return senderPeerId || 'Unknown User';
    };

    const getSenderInitial = () => {
        if (isMyMessage) return 'Y';
        const displayName = getSenderDisplayName(message.sender);
        return displayName.charAt(0).toUpperCase();
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
                    minWidth: '60px',
                }}
            >
                {!isMyMessage && (
                    <Typography
                        variant="caption"
                        color="text.secondary"
                        sx={{
                            display: 'block',
                            textAlign: 'left',
                            mb: 0.5,
                            ml: 0.5
                        }}
                    >
                        {getSenderDisplayName(message.sender)}
                    </Typography>
                )}
                <Paper
                    elevation={1}
                    sx={{
                        p: 1.5,
                        borderRadius: 2,
                        bgcolor: isMyMessage ? 'primary.main' : 'background.paper',
                        color: isMyMessage ? 'primary.contrastText' : 'text.primary',
                        wordBreak: 'break-word',
                        hyphens: 'auto',
                    }}
                >
                    <Typography variant="body1" sx={{ whiteSpace: 'pre-wrap' }}>
                        {message.text}
                    </Typography>
                </Paper>
                <Typography
                    variant="caption"
                    color="text.secondary"
                    sx={{
                        display: 'block',
                        mt: 0.5,
                        textAlign: isMyMessage ? 'right' : 'left',
                        mx: 0.5,
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