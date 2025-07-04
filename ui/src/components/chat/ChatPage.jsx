// src/components/chat/ChatPage.jsx
import React, { useState, useEffect, useRef } from 'react';
import {
    Box,
    Typography,
    TextField,
    IconButton,
    Paper,
    Divider,
    Container,
    Grid,
} from '@mui/material';
import SendIcon from '@mui/icons-material/Send';
import Sidebar from '../sidebar/Sidebar';
import websocketService from '../../services/websocket';
import { getFriends } from '../../services/api';

const ChatPage = () => {
    const [message, setMessage] = useState('');
    const [messages, setMessages] = useState([]);
    const [friends, setFriends] = useState([]);
    const [selectedFriend, setSelectedFriend] = useState(null);
    const [ownPeerId, setOwnPeerId] = useState('');
    const messagesEndRef = useRef(null);

    // Connect to WebSocket when component mounts
    useEffect(() => {
        websocketService.connect();

        // Add message listener
        const removeListener = websocketService.addMessageListener((data) => {
            if (data.type === 'DIRECT_MESSAGE') {
                const senderPeerId = data.payload.sender_peer_id;
                // Only process the message if it's from the selected friend
                // or if we're the sender (echo from server)
                const isSenderSelected = selectedFriend && senderPeerId === selectedFriend.PeerID;
                const isFromSelf = senderPeerId === ownPeerId;

                if (isSenderSelected || isFromSelf) {
                    // Check if this is our own message being echoed back
                    if (isFromSelf) {
                        // Don't add our own messages when they come back from the server
                        // They were already added when we sent them
                        return;
                    }

                    const newMessage = {
                        sender_peer_id: senderPeerId,
                        message: data.payload.message,
                        timestamp: new Date().toISOString(),
                        isOutgoing: false
                    };

                    setMessages(prev => [...prev, newMessage]);
                }
            }
        });

        // Load friends
        const loadFriends = async () => {
            try {
                const response = await getFriends();
                if (response.data) {
                    setFriends(response.data);
                    // You might want to select the first friend by default
                    if (response.data.length > 0 && !selectedFriend) {
                        setSelectedFriend(response.data[0]);
                    }
                }
            } catch (error) {
                console.error('Failed to load friends:', error);
            }
        };

        loadFriends();

        // TODO: Fetch own peer ID from an API or context
        // For now, let's assume it's stored in localStorage or similar
        setOwnPeerId(localStorage.getItem('peerID') || '');

        return () => {
            removeListener();
            websocketService.disconnect();
        };
    }, [selectedFriend, ownPeerId]);

    // Scroll to bottom when messages change
    useEffect(() => {
        messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
    }, [messages]);

    const handleSendMessage = () => {
        if (!message.trim() || !selectedFriend) return;

        const success = websocketService.sendMessage(selectedFriend.PeerID, message);

        if (success) {
            // Add the sent message to the UI
            const newMessage = {
                sender_peer_id: ownPeerId,
                message: message,
                timestamp: new Date().toISOString(),
                isOutgoing: true
            };
            setMessages(prev => [...prev, newMessage]);
            setMessage('');
        }
    };

    const formatPeerId = (peerId) => {
        if (!peerId || peerId.length < 8) return peerId;
        const first2 = peerId.substring(0, 2);
        const last6 = peerId.substring(peerId.length - 6);
        return `${first2}*${last6}`;
    };

    const getDisplayName = (friend) => {
        return friend?.display_name || formatPeerId(friend?.PeerID);
    };

    return (
        <Container maxWidth="xl" sx={{ height: '100vh', display: 'flex', p: 2 }}>
            <Grid container spacing={2} sx={{ height: '100%' }}>
                <Grid item xs={3} sx={{ height: '100%' }}>
                    <Paper sx={{ height: '100%', overflowY: 'auto' }}>
                        <Sidebar onSelectFriend={setSelectedFriend} />
                    </Paper>
                </Grid>
                <Grid item xs={9} sx={{ height: '100%' }}>
                    <Paper sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
                        {selectedFriend ? (
                            <>
                                <Box sx={{ p: 2, borderBottom: '1px solid rgba(0, 0, 0, 0.12)' }}>
                                    <Typography variant="h6">
                                        {getDisplayName(selectedFriend)}
                                    </Typography>
                                    <Typography variant="body2" color="text.secondary">
                                        {selectedFriend.IsOnline ? 'Online' : 'Offline'}
                                    </Typography>
                                </Box>

                                <Box sx={{ flexGrow: 1, p: 2, overflowY: 'auto' }}>
                                    {messages.map((msg, index) => (
                                        <Box
                                            key={index}
                                            sx={{
                                                display: 'flex',
                                                justifyContent: msg.isOutgoing ? 'flex-end' : 'flex-start',
                                                mb: 2
                                            }}
                                        >
                                            <Paper
                                                elevation={1}
                                                sx={{
                                                    p: 2,
                                                    maxWidth: '70%',
                                                    bgcolor: msg.isOutgoing ? 'primary.light' : 'grey.100',
                                                    borderRadius: 2
                                                }}
                                            >
                                                <Typography variant="body1">{msg.message}</Typography>
                                                <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 1 }}>
                                                    {new Date(msg.timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                                                </Typography>
                                            </Paper>
                                        </Box>
                                    ))}
                                    <div ref={messagesEndRef} />
                                </Box>

                                <Divider />

                                <Box sx={{ p: 2, display: 'flex', alignItems: 'center' }}>
                                    <TextField
                                        fullWidth
                                        variant="outlined"
                                        placeholder="Type a message..."
                                        value={message}
                                        onChange={(e) => setMessage(e.target.value)}
                                        onKeyPress={(e) => {
                                            if (e.key === 'Enter' && !e.shiftKey) {
                                                e.preventDefault();
                                                handleSendMessage();
                                            }
                                        }}
                                        size="small"
                                    />
                                    <IconButton
                                        color="primary"
                                        onClick={handleSendMessage}
                                        disabled={!message.trim()}
                                        sx={{ ml: 1 }}
                                    >
                                        <SendIcon />
                                    </IconButton>
                                </Box>
                            </>
                        ) : (
                            <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100%' }}>
                                <Typography variant="h6" color="text.secondary">
                                    Select a friend to start chatting
                                </Typography>
                            </Box>
                        )}
                    </Paper>
                </Grid>
            </Grid>
        </Container>
    );
};

export default ChatPage;