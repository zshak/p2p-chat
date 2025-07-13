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
import { getFriends, getGroupChatMessages } from '../../services/api';
import { getPeerId } from '../utils/userStore';
import ChatMessage from "./ChatMessage";

// Define initial number of messages to display and how many to load each time
const INITIAL_DISPLAY_COUNT = 10; // Number of messages to show initially
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
            console.log("I am: ", ownPeerId)
            if (data.type === 'DIRECT_MESSAGE' || data.type === 'GROUP_MESSAGE') {
                const { sender_peer_id, message: chatMessageText } = data.payload;

                // Determine the chat ID and type based on the message payload
                let chatId;
                // let chatType; // chatType isn't strictly needed in the message object itself

                if (data.type === 'DIRECT_MESSAGE') {
                    const { target_peer_id } = data.payload;
                    // For DMs, the chat ID is the other participant's peer ID
                    chatId = sender_peer_id === ownPeerId ? target_peer_id : sender_peer_id;
                    // chatType = 'friend';
                } else { // GROUP_MESSAGE
                    const { group_id } = data.payload;
                    chatId = group_id;
                    // chatType = 'group';
                }


                // Prevent adding our own messages echoed back if using optimistic updates
                // A more robust approach uses unique message IDs from the backend
                if (sender_peer_id === ownPeerId) {
                    console.log("Skipping echoed message from self:", chatMessageText);
                    return;
                }


                const newMessage = {
                    SenderPeerId: sender_peer_id,
                    Message: chatMessageText,
                    Time: new Date().toISOString(), // Use current time for incoming WS messages
                    isOutgoing: false, // Incoming message
                    chatId: chatId, // Store the chat ID with the message
                };

                console.log(`Adding new incoming message to chatId ${chatId}:`, newMessage);


                setMessagesByChat(prev => {
                    const chatMessages = prev[chatId] || [];
                    // Add the new message to the end
                    const updatedMessages = [...chatMessages, newMessage];

                    return {
                        ...prev,
                        [chatId]: updatedMessages,
                    };
                });

                // If the new message is for the currently selected chat AND the user is near the bottom, scroll down
                if (selectedChat && ( (selectedChat.type === 'friend' && selectedChat.PeerID === chatId) || (selectedChat.type === 'group' && selectedChat.group_id === chatId) ) ) {
                    // Check if the user is scrolled near the bottom (e.g., last 100 pixels)
                    const container = messagesContainerRef.current;
                    if (container && container.scrollHeight - container.scrollTop <= container.clientHeight + 100) {
                        // Scroll to bottom after state update and render
                        // Use a timeout to allow state update and rendering
                        setTimeout(() => {
                            messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
                        }, 50);
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
            // websocketService.disconnect(); // Manage WebSocket connection lifecycle carefully
        };
    }, [ownPeerId, selectedChat]); // Added selectedChat to dependencies to handle scrolling on new messages for active chat


    // Fetch messages when selectedChat changes
    useEffect(() => {
        const fetchMessages = async () => {
            if (selectedChat) {
                const chatId = selectedChat.type === 'friend' ? selectedChat.PeerID : selectedChat.group_id;

                // Reset displayed count and loading states for the new chat
                setDisplayedMessageCount(INITIAL_DISPLAY_COUNT);
                setLoadingOlderMessages(false); // Ensure this is false
                setLoadingInitialMessages(true); // Start initial loading


                // If messages for this chat are already in state, maybe don't refetch everything?
                // For simplicity with current API, we refetch history every time a chat is selected.
                // A more advanced approach would check if history is already loaded.


                try {
                    let fetchedMsgs = [];
                    if (selectedChat.type === 'group') {
                        const response = await getGroupChatMessages(selectedChat.group_id);
                        if (response.data && response.data.Messages) {
                            fetchedMsgs = response.data.Messages.map(msg => ({
                                SenderPeerId: msg.SenderPeerId,
                                Message: msg.Message,
                                Time: msg.Time, // Use the time from the API for history
                                isOutgoing: msg.SenderPeerId === ownPeerId,
                                chatId: chatId,
                            }));
                            console.log(`Fetched ${fetchedMsgs.length} historical messages for group ${chatId}`);
                        }
                    } else if (selectedChat.type === 'friend') {
                        // TODO: Implement fetching historical direct messages if API exists
                        console.warn("Fetching historical direct messages is not implemented yet.");
                        // If you fetch DMs, map them similarly and add chatId: chatId
                    }

                    // --- Update State with Fetched Messages ---
                    // This replaces any previous messages for this chat in state.
                    // If WebSocket messages arrived before the fetch, the WS listener
                    // would have added them *after* this fetch completes due to state updates.
                    // Sorting is important to ensure correct chronological order.
                    const sortedFetchedMsgs = fetchedMsgs.sort((a, b) => new Date(a.Time).getTime() - new Date(b.Time).getTime());

                    setMessagesByChat(prev => ({
                        ...prev,
                        [chatId]: sortedFetchedMsgs, // Store ALL fetched messages
                    }));
                    console.log(`Updated messagesByChat for chat ${chatId} with ${sortedFetchedMsgs.length} fetched messages.`);

                    // --- End State Update ---


                } catch (error) {
                    console.error('Failed to fetch messages for chat:', chatId, error);
                    // Ensure the chat's message array exists even if fetching fails
                    setMessagesByChat(prev => ({ ...prev, [chatId]: prev[chatId] || [] }));
                } finally {
                    setLoadingInitialMessages(false); // Finish initial loading

                    // Scroll to bottom after initial load/fetch
                    // Use a timeout to allow state update and rendering
                    setTimeout(() => {
                        messagesEndRef.current?.scrollIntoView({ behavior: 'smooth', block: 'end' });
                    }, 150); // A slightly longer delay might be needed depending on rendering speed
                }
            } else {
                // If no chat selected, clear messages (optional)
                // setMessagesByChat({});
            }
        };

        fetchMessages();

    }, [selectedChat, ownPeerId]); // Depend on selectedChat and ownPeerId


    // Effect for handling scroll to load more
    useEffect(() => {
        const container = messagesContainerRef.current;
        if (!container || !selectedChat) return; // Only add listener if container and chat are selected

        const handleScroll = () => {
            // Check if scrolled near the top (e.g., within 100 pixels of the top)
            if (container.scrollTop < 100 && !loadingOlderMessages) {
                const chatId = selectedChat.type === 'friend' ? selectedChat.PeerID : selectedChat.group_id;
                const allMessages = messagesByChat[chatId] || [];

                // Check if there are more messages to potentially load (i.e., total messages exceed currently displayed)
                if (allMessages.length > displayedMessageCount) {
                    setLoadingOlderMessages(true); // Indicate loading
                    console.log("Scrolled to top, attempting to load more messages.");

                    // Store current scroll position before loading more to try and maintain it
                    const currentScrollHeight = container.scrollHeight;

                    // Increase the number of messages to display
                    setDisplayedMessageCount(prevCount => {
                        const newCount = prevCount + MESSAGES_TO_LOAD_MORE;
                        console.log(`Increasing displayed message count from ${prevCount} to ${newCount}`);
                        return newCount;
                    });

                    // After state updates and renders the new messages, adjust scroll position
                    // Use a timeout to wait for rendering
                    setTimeout(() => {
                        const newScrollHeight = container.scrollHeight;
                        // Adjust scroll top to keep the view relatively stable
                        container.scrollTop = newScrollHeight - currentScrollHeight;
                        setLoadingOlderMessages(false); // Finish loading indicator
                        console.log("Scroll position adjusted after loading more.");
                    }, 50); // Small delay


                } else {
                    // No more older messages to load
                    console.log("Scrolled to top, but no more older messages available in state.");
                }
            }
        };

        container.addEventListener('scroll', handleScroll);

        return () => {
            container.removeEventListener('scroll', handleScroll);
        };
    }, [selectedChat, messagesByChat, displayedMessageCount, loadingOlderMessages]); // Dependencies: selectedChat, messagesByChat (to react to new messages being added), displayedCount, loading state


    const handleSendMessage = () => {
        if (!message.trim() || !selectedChat) return;

        const messageToSend = message;
        setMessage(''); // Clear input immediately

        const chatId = selectedChat.type === 'friend' ? selectedChat.PeerID : selectedChat.group_id;
        let success = false;

        if (selectedChat.type === 'friend') {
            success = websocketService.sendMessage(selectedChat.PeerID, messageToSend);
        } else if (selectedChat.type === 'group') {
            success = websocketService.sendGroupMessage(selectedChat.group_id, messageToSend);
        }

        if (success) {
            // Optimistically add the sent message to the UI
            const newMessage = {
                SenderPeerId: ownPeerId,
                Message: messageToSend,
                Time: new Date().toISOString(), // Use current time for optimistic update
                isOutgoing: true,
                chatId: chatId, // Add chat ID
            };

            setMessagesByChat(prev => {
                const chatMessages = prev[chatId] || [];
                return {
                    ...prev,
                    [chatId]: [...chatMessages, newMessage] // Add to the end
                };
            });

            // After sending a message, ensure the full count is displayed AND scroll to the bottom
            setDisplayedMessageCount(prevCount => {
                const allMessages = messagesByChat[chatId] || [];
                // If the number of messages is now more than the current displayed count + 1 (the new message),
                // update the displayed count to show the new message and potentially more recent ones
                if (allMessages.length + 1 > prevCount) {
                    return allMessages.length + 1; // Show all messages including the new one
                }
                return prevCount; // Otherwise, maintain the current displayed count
            });


            // Scroll to bottom after sending and updating displayed count
            // Use a timeout to allow state update and rendering
            setTimeout(() => {
                messagesEndRef.current?.scrollIntoView({ behavior: 'smooth', block: 'end' });
            }, 50); // Small delay

        } else {
            console.error("Failed to send message.");
            // TODO: Show an error to the user
        }
    };

    // Determine messages for the currently selected chat from the state
    const currentChatId = selectedChat?.type === 'friend' ? selectedChat.PeerID : selectedChat?.group_id;
    const allMessagesForSelectedChat = messagesByChat[currentChatId] || [];

    // --- Limit Rendering Here ---
    // Calculate the starting index for the slice based on the desired displayed count
    const startIndex = Math.max(0, allMessagesForSelectedChat.length - displayedMessageCount);
    // Slice the messages to display from the beginning based on the calculated start index
    const messagesToDisplay = allMessagesForSelectedChat.slice(startIndex);
    // --- End Limit Rendering ---

    // Calculate how many older messages are not being displayed
    const olderMessagesCount = allMessagesForSelectedChat.length - messagesToDisplay.length;


    // Pass friends list and the setSelectedChat function to Sidebar
    return (
        <Container maxWidth="xl" sx={{ height: '100vh', display: 'flex', p: 2 }}>
            <Grid container spacing={2} sx={{ height: '100%' }}>
                <Grid item xs={3} sx={{ height: '100%' }}>
                    <Paper sx={{ height: '100%', display: 'flex', flexDirection: 'column', overflowY: 'auto' }}>
                        {/* Pass the setSelectedChat callback down */}
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

                                {/* Add the ref to the scrollable container */}
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
                                            // Use a more stable key if possible (like a message ID from backend)
                                            // Fallback to index + chat ID as messages are unique per chat
                                            key={`${msg.chatId}-${msg.Time}-${msg.SenderPeerId}-${index}`} // More unique key
                                            message={{
                                                sender: msg.SenderPeerId,
                                                text: msg.Message,
                                                timestamp: msg.Time,
                                            }}
                                            currentUser={{ peerId: ownPeerId }}
                                            // Pass chat type or members if needed by ChatMessage for sender names
                                            // chatMembers={selectedChat.type === 'group' ? selectedChat.members : undefined}
                                        />
                                    ))}
                                    <div ref={messagesEndRef} /> {/* Element to scroll into view */}
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
                                        disabled={loadingInitialMessages} // Disable input while initial messages load
                                    />
                                    <IconButton
                                        color="primary"
                                        onClick={handleSendMessage}
                                        disabled={!message.trim() || loadingInitialMessages} // Disable if empty or loading
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