// src/components/chat/ChatPage.jsx
import React, { useState, useEffect, useRef, useCallback } from 'react';
import {
    Box,
    Typography,
    TextField,
    IconButton,
    Paper,
    Divider,
    Container,
    Grid,
    CircularProgress // Import CircularProgress for loading indicator
} from '@mui/material';
import SendIcon from '@mui/icons-material/Send';
import Sidebar from '../sidebar/Sidebar';
import websocketService from '../../services/websocket';
import { getFriends, getGroupChatMessages, getChatMessages } from '../../services/api';
import { getPeerId } from '../utils/userStore';
import ChatMessage from "./ChatMessage";

// Define initial number of messages to display and how many to load each time
const INITIAL_DISPLAY_COUNT = 10; // Number of messages to show initially (increased for better default view)
const MESSAGES_TO_LOAD_MORE = 10; // Number of older messages to load when scrolling up

const ChatPage = () => {
    const [message, setMessage] = useState('');
    // Use messagesByChat to store ALL messages fetched/received for each chat
    const [messagesByChat, setMessagesByChat] = useState({});
    const [selectedChat, setSelectedChat] = useState(null);
    const [ownPeerId, setOwnPeerId] = useState('');
    const messagesEndRef = useRef(null);
    const messagesContainerRef = useRef(null); // Ref for the scrollable message container
    const [friends, setFriends] = useState([]);

    // State to track the number of messages to DISPLAY for the CURRENTLY selected chat
    const [displayedMessageCount, setDisplayedMessageCount] = useState(INITIAL_DISPLAY_COUNT);
    // State to track if we are currently loading older messages (to prevent multiple fetches/state updates)
    const [loadingOlderMessages, setLoadingOlderMessages] = useState(false);
    // State to track initial historical message loading
    const [loadingInitialMessages, setLoadingInitialMessages] = useState(false);


    // Connect to WebSocket and add listener
    useEffect(() => {
        websocketService.connect();

        const removeListener = websocketService.addMessageListener((data) => {
            console.log("WebSocket message received:", data);
            console.log("My Peer ID is: ", ownPeerId); // Added for debugging ownPeerId

            // IMPORTANT: Check if ownPeerId is available before processing messages that depend on it
            // If it's not set yet, we can't determine if message is outgoing or from correct chat
            if (!ownPeerId) {
                console.warn("Received WebSocket message before ownPeerId was set. Message:", data);
                return;
            }


            if (data.type === 'DIRECT_MESSAGE' || data.type === 'GROUP_MESSAGE') {
                const { sender_peer_id, message: chatMessageText } = data.payload;

                let chatId;
                if (data.type === 'DIRECT_MESSAGE') {
                    const { target_peer_id } = data.payload;
                    // For DMs, the chat ID is the other participant's peer ID
                    chatId = sender_peer_id === ownPeerId ? target_peer_id : sender_peer_id;
                } else { // GROUP_MESSAGE
                    const { group_id } = data.payload;
                    chatId = group_id;
                }

                const newMessage = {
                    SenderPeerId: sender_peer_id,
                    Message: chatMessageText,
                    // Use server timestamp if available, otherwise current.
                    // For direct messages, the backend provides "Time", for group it's implicit (or could be in payload if server sends).
                    Time: data.payload.Time || new Date().toISOString(),
                    isOutgoing: sender_peer_id === ownPeerId, // Determine if it's our message
                    chatId: chatId,
                };

                console.log(`Adding new incoming/echoed message to chatId ${chatId}:`, newMessage);

                setMessagesByChat(prev => {
                    const chatMessages = prev[chatId] || [];
                    const updatedMessages = [...chatMessages, newMessage];
                    return {
                        ...prev,
                        [chatId]: updatedMessages,
                    };
                });

                // If the new message is for the currently selected chat AND the user is near the bottom, scroll down
                if (selectedChat && ( (selectedChat.type === 'friend' && selectedChat.PeerID === chatId) || (selectedChat.type === 'group' && selectedChat.group_id === chatId) ) ) {
                    const container = messagesContainerRef.current;
                    if (container) {
                        const isAtBottom = container.scrollHeight - container.scrollTop <= container.clientHeight + 100;
                        if (isAtBottom) {
                            setTimeout(() => {
                                messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
                            }, 50);
                        }
                    }
                }
            }
        });

        // Load friends (needed for CreateGroupChat modal and potentially resolving names)
        const loadFriends = async () => {
            try {
                const response = await getFriends();
                if (response.data) {
                    setFriends(response.data);
                }
            } catch (error) {
                console.error('Failed to load friends:', error);
            }
        };

        loadFriends();

        // Fetch own peer ID
        setOwnPeerId(getPeerId());

        return () => {
            removeListener();
        };
    }, [ownPeerId, selectedChat]);


    // Fetch messages when selectedChat changes
    useEffect(() => {
        const fetchMessages = async () => {
            if (selectedChat) {
                const chatId = selectedChat.type === 'friend' ? selectedChat.PeerID : selectedChat.group_id;

                setDisplayedMessageCount(INITIAL_DISPLAY_COUNT);
                setLoadingOlderMessages(false);
                setLoadingInitialMessages(true);

                try {
                    let fetchedMsgs = [];
                    if (selectedChat.type === 'group') {
                        const response = await getGroupChatMessages(selectedChat.group_id);
                        if (response.data && response.data.Messages) {
                            fetchedMsgs = response.data.Messages.map(msg => ({
                                SenderPeerId: msg.SenderPeerId,
                                Message: msg.Message,
                                Time: msg.Time,
                                isOutgoing: msg.SenderPeerId === ownPeerId,
                                chatId: chatId,
                            }));
                            console.log(`Fetched ${fetchedMsgs.length} historical messages for group ${chatId}`);
                        }
                    } else if (selectedChat.type === 'friend') {
                        // Fetch historical direct messages ===
                        const response = await getChatMessages(selectedChat.PeerID); // Pass the friend's peer_id
                        if (response.data && response.data.Messages) {
                            fetchedMsgs = response.data.Messages.map(msg => ({
                                // For direct messages, determine SenderPeerId based on IsOutgoing
                                SenderPeerId: msg.IsOutgoing ? ownPeerId : selectedChat.PeerID,
                                Message: msg.Message,
                                Time: msg.SendTime,
                                isOutgoing: msg.IsOutgoing,
                                chatId: chatId,
                            }));
                            console.log(`Fetched ${fetchedMsgs.length} historical messages for friend ${chatId}`);
                        }
                    }

                    const sortedFetchedMsgs = fetchedMsgs.sort((a, b) => new Date(a.Time).getTime() - new Date(b.Time).getTime());

                    setMessagesByChat(prev => ({
                        ...prev,
                        [chatId]: sortedFetchedMsgs,
                    }));
                    console.log(`Updated messagesByChat for chat ${chatId} with ${sortedFetchedMsgs.length} fetched messages.`);

                } catch (error) {
                    console.error('Failed to fetch messages for chat:', chatId, error);
                    setMessagesByChat(prev => ({ ...prev, [chatId]: prev[chatId] || [] }));
                } finally {
                    setLoadingInitialMessages(false);

                    setTimeout(() => {
                        messagesEndRef.current?.scrollIntoView({ behavior: 'smooth', block: 'end' });
                    }, 150);
                }
            } else {
                // If no chat selected, clear messages (optional)
            }
        };

        fetchMessages();

    }, [selectedChat, ownPeerId]);


    // Effect for handling scroll to load more
    useEffect(() => {
        const container = messagesContainerRef.current;
        if (!container || !selectedChat) return;

        const handleScroll = () => {
            if (container.scrollTop < 100 && !loadingOlderMessages) {
                const chatId = selectedChat.type === 'friend' ? selectedChat.PeerID : selectedChat.group_id;
                const allMessages = messagesByChat[chatId] || [];

                if (allMessages.length > displayedMessageCount) {
                    setLoadingOlderMessages(true);
                    console.log("Scrolled to top, attempting to load more messages.");

                    const currentScrollHeight = container.scrollHeight;

                    setDisplayedMessageCount(prevCount => {
                        const newCount = prevCount + MESSAGES_TO_LOAD_MORE;
                        console.log(`Increasing displayed message count from ${prevCount} to ${newCount}`);
                        return newCount;
                    });

                    setTimeout(() => {
                        const newScrollHeight = container.scrollHeight;
                        container.scrollTop = newScrollHeight - currentScrollHeight;
                        setLoadingOlderMessages(false);
                        console.log("Scroll position adjusted after loading more.");
                    }, 50);

                } else {
                    console.log("Scrolled to top, but no more older messages available in state.");
                }
            }
        };

        container.addEventListener('scroll', handleScroll);

        return () => {
            container.removeEventListener('scroll', handleScroll);
        };
    }, [selectedChat, messagesByChat, displayedMessageCount, loadingOlderMessages]);


    const handleSendMessage = () => {
        if (!message.trim() || !selectedChat) return;

        const messageToSend = message;
        setMessage('');

        if (selectedChat.type === 'friend') {
            websocketService.sendMessage(selectedChat.PeerID, messageToSend);
        } else if (selectedChat.type === 'group') {
            websocketService.sendGroupMessage(selectedChat.group_id, messageToSend);
        }
    };

    const currentChatId = selectedChat?.type === 'friend' ? selectedChat.PeerID : selectedChat?.group_id;
    const allMessagesForSelectedChat = messagesByChat[currentChatId] || [];

    const startIndex = Math.max(0, allMessagesForSelectedChat.length - displayedMessageCount);
    const messagesToDisplay = allMessagesForSelectedChat.slice(startIndex);

    const olderMessagesCount = allMessagesForSelectedChat.length - messagesToDisplay.length;


    return (
        <Container maxWidth="xl" sx={{ height: '100vh', display: 'flex', p: 2 }}>
            <Grid container spacing={2} sx={{ height: '100%' }}>
                <Grid item xs={3} sx={{ height: '100%' }}>
                    <Paper sx={{ height: '100%', display: 'flex', flexDirection: 'column', overflowY: 'auto' }}>
                        <Sidebar onSelectChat={setSelectedChat} friends={friends} />
                    </Paper>
                </Grid>
                <Grid item xs={9} sx={{ height: '100%' }}>
                    <Paper sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
                        {selectedChat ? (
                            <>
                                <Box sx={{ p: 2, borderBottom: '1px solid rgba(0, 0, 0, 0.12)' }}>
                                    <Typography variant="h6">
                                        {selectedChat.type === 'friend' ? selectedChat.display_name || selectedChat.PeerID : selectedChat.name || `Group (${selectedChat.members.length})`}
                                    </Typography>
                                    <Typography variant="body2" color="text.secondary">
                                        {selectedChat.type === 'friend' ? (selectedChat.IsOnline ? 'Online' : 'Offline') : `${selectedChat.members.length} members`}
                                    </Typography>
                                </Box>

                                <Box
                                    ref={messagesContainerRef}
                                    sx={{ flexGrow: 1, p: 2, overflowY: 'auto', display: 'flex', flexDirection: 'column' }}
                                >
                                    {/* Show loading indicator for initial fetch */}
                                    {loadingInitialMessages && allMessagesForSelectedChat.length === 0 && (
                                        <Box sx={{ display: 'flex', justifyContent: 'center', my: 2 }}>
                                            <CircularProgress size={20} />
                                        </Box>
                                    )}

                                    {/* Show loading indicator for older messages */}
                                    {loadingOlderMessages && (
                                        <Box sx={{ display: 'flex', justifyContent: 'center', my: 1 }}>
                                            <CircularProgress size={20} />
                                        </Box>
                                    )}

                                    {/* Indicator for older messages not shown */}
                                    {olderMessagesCount > 0 && !loadingOlderMessages && (
                                        <Box sx={{ textAlign: 'center', mb: 2 }}>
                                            <Typography variant="body2" color="text.secondary">
                                                Scroll up to load {Math.min(olderMessagesCount, MESSAGES_TO_LOAD_MORE)} older messages.
                                            </Typography>
                                        </Box>
                                    )}

                                    {/* Map over the sliced array of messages to display */}
                                    {messagesToDisplay.map((msg, index) => (
                                        <ChatMessage
                                            key={`${msg.chatId}-${msg.Time}-${msg.SenderPeerId}-${index}`}
                                            message={{
                                                sender: msg.SenderPeerId,
                                                text: msg.Message,
                                                timestamp: msg.Time,
                                            }}
                                            currentUser={{ peerId: ownPeerId }}
                                        />
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
                                        disabled={loadingInitialMessages}
                                    />
                                    <IconButton
                                        color="primary"
                                        onClick={handleSendMessage}
                                        disabled={!message.trim() || loadingInitialMessages}
                                        sx={{ ml: 1 }}
                                    >
                                        <SendIcon />
                                    </IconButton>
                                </Box>
                            </>
                        ) : (
                            <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100%' }}>
                                <Typography variant="h6" color="text.secondary">
                                    Select a chat to start messaging
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