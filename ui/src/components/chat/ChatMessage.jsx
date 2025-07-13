// src/components/chat/ChatMessage.jsx
import React from 'react';
import {
    Box,
    Paper,
    Typography,
    Avatar,
} from '@mui/material';
import { getPeerId } from '../utils/userStore'; // Assuming this utility exists

const ChatMessage = ({ message, currentUser, chatMembers }) => {
    const ownPeerId = getPeerId(); // Get current user's peer ID
    const isMyMessage = message.sender === 'me' || message.sender === ownPeerId; // Compare with ownPeerId

    const formatTime = (timestamp) => {
        if (!timestamp) return '';
        const date = new Date(timestamp);
        // Check if the date is valid before formatting
        if (isNaN(date.getTime())) {
            console.error("Invalid timestamp:", timestamp);
            return 'Invalid Date';
        }
        return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    };

    // Function to get sender's display name or a truncated peer ID for group chats
    const getSenderDisplayName = (senderPeerId) => {
        // In a real app, you'd look up the sender's name from your friends/members list
        // For simplicity here, we'll just use a truncated peer ID or a default
        if (senderPeerId === ownPeerId) return 'You';

        // If chatMembers are provided (for group chats), try to find the member
        // and return their display name if available. (requires chatMembers prop to be fully implemented in ChatPage)
        // const member = chatMembers?.find(m => m.PeerID === senderPeerId);
        // if (member?.display_name) return member.display_name;

        // Fallback to truncated peer ID
        if (senderPeerId && senderPeerId.length > 8) {
            return `${senderPeerId.substring(0, 4)}...${senderPeerId.substring(senderPeerId.length - 4)}`;
        }
        return senderPeerId || 'Unknown User';
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
                    {/* You might want to get the initial from the sender's name in group chats */}
                    {message.sender ? getSenderDisplayName(message.sender).charAt(0).toUpperCase() : 'U'}
                </Avatar>
            )}
            <Box
                sx={{
                    maxWidth: '70%',
                    // minWidth: '100px', // Consider removing or adjusting minWidth
                }}
            >
                {!isMyMessage && ( // Display sender name in group chats (if not my message)
                    <Typography variant="caption" color="text.secondary" sx={{ display: 'block', textAlign: 'left', mb: 0.5 }}>
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
                        wordBreak: 'break-word', // Ensure long words break
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
                    {/* You might want to get the initial from the current user's name */}
                    Y {/* Assuming 'Y' for 'You' or replace with actual user initial */}
                </Avatar>
            )}
        </Box>
    );
};

export default ChatMessage;