import React, { useState, useEffect, useRef, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import {
    AppBar,
    Avatar,
    Box,
    CircularProgress,
    Container,
    Drawer,
    IconButton,
    Toolbar,
    Typography,
    useMediaQuery,
    useTheme,
    TextField,
    Paper
} from '@mui/material';
import MenuIcon from '@mui/icons-material/Menu';
import CloseIcon from '@mui/icons-material/Close';
import SendIcon from '@mui/icons-material/Send';
import ChatMessage from './ChatMessage';
import Sidebar from '../sidebar/Sidebar';
import { DAEMON_STATES } from '../utils/constants';
import { checkStatus, getFriends, getGroupChatMessages, getChatMessages, getDisplayNameAPI} from "../../services/api.js";
import chatIcon from '../../../public/icon.svg';
import websocketService from '../../services/websocket';

// Mock getPeerId function - replace with your actual implementation
const getPeerId = () => {
    const peerId = localStorage.getItem('userPeerId') || localStorage.getItem('peerID') || 'mock-peer-id';
    return peerId.trim(); // Ensure no whitespace
};

const MESSAGES_PER_PAGE = 10;

function ChatPage() {
    const navigate = useNavigate();
    const theme = useTheme();
    const isMobile = useMediaQuery(theme.breakpoints.down('md'));
    const [status, setStatus] = useState(null);
    const [loading, setLoading] = useState(true);
    const [drawerOpen, setDrawerOpen] = useState(!isMobile);
    const messagesEndRef = useRef(null);
    const messagesContainerRef = useRef(null);
    const loadMoreTriggerRef = useRef(null);
    const [refreshTrigger, setRefreshTrigger] = useState(0);

    // Chat state
    const [message, setMessage] = useState('');
    const [messagesByChat, setMessagesByChat] = useState({});
    const [selectedChat, setSelectedChat] = useState(null);
    const [ownPeerId, setOwnPeerId] = useState('');
    const [friends, setFriends] = useState([]);
    const [isLoadingMessages, setIsLoadingMessages] = useState(false);
    const [isLoadingMoreMessages, setIsLoadingMoreMessages] = useState(false);

    // Display name state
    const [selectedChatDisplayName, setSelectedChatDisplayName] = useState('');

    // Pagination state for each chat
    const [chatPaginationState, setChatPaginationState] = useState({});

    // Function to update display names when they change
    const handleDisplayNameUpdate = useCallback((entityId, entityType, newDisplayName) => {
        // Update friends list if it's a friend
        if (entityType === 'friend') {
            setFriends(prev => prev.map(friend =>
                friend.PeerID === entityId
                    ? { ...friend, display_name: newDisplayName || undefined }
                    : friend
            ));
        }

        // Update selected chat display name if it matches
        if (selectedChat) {
            const currentEntityId = selectedChat.type === 'friend' ? selectedChat.PeerID : selectedChat.group_id;
            if (currentEntityId === entityId && selectedChat.type === entityType) {
                if (newDisplayName) {
                    setSelectedChatDisplayName(newDisplayName);
                } else {
                    // Reset to default name
                    if (selectedChat.type === 'friend') {
                        setSelectedChatDisplayName(selectedChat.PeerID);
                    } else {
                        setSelectedChatDisplayName(selectedChat.name || `Group (${selectedChat.members?.length || 0})`);
                    }
                }
            }
        }
    }, [selectedChat]);

    // Load selected chat display name
    const loadSelectedChatDisplayName = useCallback(async (chat) => {
        if (!chat) {
            setSelectedChatDisplayName('');
            return;
        }

        const entityId = chat.type === 'friend' ? chat.PeerID : chat.group_id;
        const entityType = chat.type;

        // First check if we have the display name in the chat object (for friends)
        if (chat.type === 'friend' && chat.display_name) {
            setSelectedChatDisplayName(chat.display_name);
            return;
        }

        try {
            const displayNameData = await getDisplayNameAPI(entityId, entityType);
            if (displayNameData && displayNameData.display_name) {
                setSelectedChatDisplayName(displayNameData.display_name);
            } else {
                // Use default name
                if (chat.type === 'friend') {
                    setSelectedChatDisplayName(chat.PeerID);
                } else {
                    setSelectedChatDisplayName(chat.name || `Group (${chat.members?.length || 0})`);
                }
            }
        } catch (error) {
            // Display name not found, use default
            if (chat.type === 'friend') {
                setSelectedChatDisplayName(chat.PeerID);
            } else {
                setSelectedChatDisplayName(chat.name || `Group (${chat.members?.length || 0})`);
            }
        }
    }, []);

    // Initialize component
    useEffect(() => {
        const initializeChat = async () => {
            try {
                const response = await checkStatus();
                setStatus(response.data.state);

                if (response.data.state !== DAEMON_STATES.RUNNING) {
                    navigate('/login');
                    return;
                }

                const peerId = response.data.peer_id || getPeerId();
                setOwnPeerId(peerId);

                // Also store it in localStorage for consistency
                if (response.data.peer_id) {
                    localStorage.setItem('userPeerId', response.data.peer_id);
                    localStorage.setItem('peerID', response.data.peer_id);
                }

                try {
                    const friendsResponse = await getFriends();
                    if (friendsResponse.data) {
                        setFriends(friendsResponse.data);
                    }
                } catch (error) {
                    console.error('Failed to load friends:', error);
                    setFriends([]);
                }

                setLoading(false);
            } catch (err) {
                console.error('Initialization failed:', err);
                navigate('/login');
            }
        };

        initializeChat();
    }, [navigate]);

    // Auto-refresh status
    useEffect(() => {
        const interval = setInterval(async () => {
            try {
                if (!document.hidden) {
                    const statusResponse = await checkStatus();
                    setStatus(statusResponse.data.state);

                    if (statusResponse.data.state !== DAEMON_STATES.RUNNING) {
                        navigate('/login');
                        return;
                    }

                    setRefreshTrigger(prev => prev + 1);
                }
            } catch (err) {
                console.error('Status check failed:', err);
            }
        }, 5000);

        return () => clearInterval(interval);
    }, [navigate]);

    // Smooth scroll to bottom function
    const scrollToBottom = useCallback((behavior = 'smooth', force = false) => {
        if (messagesEndRef.current) {
            // For initial load, use scrollTop to ensure we're at the bottom
            if (force && messagesContainerRef.current) {
                messagesContainerRef.current.scrollTop = messagesContainerRef.current.scrollHeight;
            } else {
                messagesEndRef.current.scrollIntoView({
                    behavior,
                    block: 'end'
                });
            }
        }
    }, []);

    useEffect(() => {
        // Connect to WebSocket
        websocketService.connect();

        // Add message listener
        const removeListener = websocketService.addMessageListener((data) => {
            console.log("WebSocket message received:", data);

            // Check if ownPeerId is available before processing messages
            if (!ownPeerId) {
                console.warn("Received WebSocket message before ownPeerId was set. Message:", data);
                return;
            }

            if (data.type === 'DIRECT_MESSAGE' || data.type === 'GROUP_MESSAGE') {
                const { sender_peer_id, message: chatMessageText, target_peer_id, group_id } = data.payload;

                console.log("Processing WebSocket message:", {
                    type: data.type,
                    sender: sender_peer_id,
                    message: chatMessageText,
                    ownPeerId: ownPeerId,
                    isMyMessage: sender_peer_id === ownPeerId
                });

                let chatId;
                if (data.type === 'DIRECT_MESSAGE') {
                    // For DMs, the chat ID is the other participant's peer ID
                    // If I sent the message, chat ID is the target
                    // If I received the message, chat ID is the sender
                    chatId = sender_peer_id.trim() === ownPeerId.trim() ? target_peer_id : sender_peer_id;
                } else { // GROUP_MESSAGE
                    chatId = group_id;
                }

                const newMessage = {
                    SenderPeerId: sender_peer_id,
                    Message: chatMessageText,
                    Time: data.payload.Time || new Date().toISOString(),
                    isOutgoing: sender_peer_id.trim() === ownPeerId.trim(),
                    chatId: chatId,
                };

                console.log(`Adding new incoming/echoed message to chatId ${chatId}:`, newMessage);

                // Update BOTH states - pagination state AND messagesByChat
                setChatPaginationState(prev => {
                    const chatState = prev[chatId];
                    if (chatState) {
                        return {
                            ...prev,
                            [chatId]: {
                                ...chatState,
                                allMessages: [...chatState.allMessages, newMessage],
                                displayedMessages: [...chatState.displayedMessages, newMessage]
                            }
                        };
                    } else {
                        // If no chat state exists, create it
                        return {
                            ...prev,
                            [chatId]: {
                                allMessages: [newMessage],
                                displayedMessages: [newMessage],
                                hasMore: false,
                                currentPage: 1
                            }
                        };
                    }
                });

                // IMPORTANT: Also update messagesByChat
                setMessagesByChat(prev => {
                    const chatMessages = prev[chatId] || [];
                    return {
                        ...prev,
                        [chatId]: [...chatMessages, newMessage]
                    };
                });

                // If the new message is for the currently selected chat AND the user is near the bottom, scroll down
                if (selectedChat &&
                    ((selectedChat.type === 'friend' && selectedChat.PeerID === chatId) ||
                        (selectedChat.type === 'group' && selectedChat.group_id === chatId))) {
                    const container = messagesContainerRef.current;
                    if (container) {
                        const isAtBottom = container.scrollHeight - container.scrollTop <= container.clientHeight + 100;
                        if (isAtBottom) {
                            setTimeout(() => scrollToBottom('smooth'), 50);
                        }
                    }
                }
            }
        });

        return () => {
            removeListener();
        };
    }, [ownPeerId, selectedChat, scrollToBottom]);

    // Load all messages for a chat and set up pagination
    const loadAllMessages = useCallback(async (chat) => {
        if (!chat || !ownPeerId) return [];

        try {
            let fetchedMsgs = [];

            if (chat.type === 'group') {
                const response = await getGroupChatMessages(chat.group_id);
                if (response.data && response.data.Messages) {
                    fetchedMsgs = response.data.Messages.map(msg => ({
                        SenderPeerId: msg.SenderPeerId,
                        Message: msg.Message,
                        Time: msg.Time,
                        isOutgoing: msg.SenderPeerId === ownPeerId,
                        chatId: chat.group_id,
                    }));
                }
            } else if (chat.type === 'friend') {
                const response = await getChatMessages(chat.PeerID);
                if (response.data && response.data.Messages) {
                    fetchedMsgs = response.data.Messages.map(msg => ({
                        SenderPeerId: msg.IsOutgoing ? ownPeerId : chat.PeerID,
                        Message: msg.Message,
                        Time: msg.SendTime,
                        isOutgoing: msg.IsOutgoing,
                        chatId: chat.PeerID,
                    }));
                }
            }

            // Sort messages by time (oldest first)
            return fetchedMsgs.sort((a, b) => new Date(a.Time).getTime() - new Date(b.Time).getTime());

        } catch (error) {
            console.error('Failed to load messages:', error);
            return [];
        }
    }, [ownPeerId]);

    // Get the last N messages for initial display
    const getLastNMessages = useCallback((messages, n) => {
        return messages.slice(-n);
    }, []);

    // Load more messages (previous messages)
    const loadMoreMessages = useCallback((chatId) => {
        setChatPaginationState(prev => {
            const chatState = prev[chatId];
            if (!chatState || !chatState.hasMore) return prev;

            const currentDisplayCount = chatState.displayedMessages.length;
            const totalMessages = chatState.allMessages.length;
            const messagesToLoad = Math.min(MESSAGES_PER_PAGE, totalMessages - currentDisplayCount);

            if (messagesToLoad <= 0) {
                return {
                    ...prev,
                    [chatId]: {
                        ...chatState,
                        hasMore: false
                    }
                };
            }

            // Get messages from the beginning that aren't displayed yet
            const startIndex = totalMessages - currentDisplayCount - messagesToLoad;
            const newMessages = chatState.allMessages.slice(startIndex, totalMessages - currentDisplayCount);

            return {
                ...prev,
                [chatId]: {
                    ...chatState,
                    displayedMessages: [...newMessages, ...chatState.displayedMessages],
                    hasMore: startIndex > 0,
                    currentPage: chatState.currentPage + 1
                }
            };
        });
    }, []);

    // Handle chat selection
    const handleSelectChat = useCallback(async (chat) => {
        console.log('Selected chat:', chat);

        if (selectedChat?.PeerID === chat.PeerID && selectedChat?.group_id === chat.group_id) {
            return;
        }

        setSelectedChat(chat);
        setMessage('');

        await loadSelectedChatDisplayName(chat);
        const chatId = chat.type === 'friend' ? chat.PeerID : chat.group_id;

        // Check if we already have pagination state for this chat
        if (chatPaginationState[chatId]) {
            // Just scroll to bottom without reloading
            setTimeout(() => scrollToBottom('auto', true), 50);
            return;
        }

        setIsLoadingMessages(true);

        try {
            // Load all messages from backend
            const allMessages = await loadAllMessages(chat);

            // Get the last 20 messages for initial display
            const initialMessages = getLastNMessages(allMessages, MESSAGES_PER_PAGE);

            // Set up pagination state
            setChatPaginationState(prev => ({
                ...prev,
                [chatId]: {
                    allMessages: allMessages,
                    displayedMessages: initialMessages,
                    hasMore: allMessages.length > MESSAGES_PER_PAGE,
                    currentPage: 1
                }
            }));

            // Update the legacy messagesByChat for compatibility
            setMessagesByChat(prev => ({
                ...prev,
                [chatId]: initialMessages,
            }));

            // Scroll to bottom after messages are loaded
            setTimeout(() => scrollToBottom('auto', true), 100);
            // Additional fallback scroll
            setTimeout(() => {
                if (messagesContainerRef.current) {
                    messagesContainerRef.current.scrollTop = messagesContainerRef.current.scrollHeight;
                }
            }, 200);

        } catch (error) {
            console.error('Failed to load messages:', error);
            setChatPaginationState(prev => ({
                ...prev,
                [chatId]: {
                    allMessages: [],
                    displayedMessages: [],
                    hasMore: false,
                    currentPage: 1
                }
            }));
            setMessagesByChat(prev => ({...prev, [chatId]: []}));
        } finally {
            setIsLoadingMessages(false);
        }
    }, [selectedChat, chatPaginationState, loadAllMessages, getLastNMessages, scrollToBottom, loadSelectedChatDisplayName]);

    // Update messagesByChat when pagination state changes and ensure scroll to bottom
    useEffect(() => {
        if (selectedChat) {
            const chatId = selectedChat.type === 'friend' ? selectedChat.PeerID : selectedChat.group_id;
            const chatState = chatPaginationState[chatId];

            if (chatState) {
                setMessagesByChat(prev => ({
                    ...prev,
                    [chatId]: chatState.displayedMessages
                }));

                // If this is the initial load (first page), scroll to bottom
                if (chatState.currentPage === 1 && chatState.displayedMessages.length > 0) {
                    setTimeout(() => {
                        if (messagesContainerRef.current) {
                            messagesContainerRef.current.scrollTop = messagesContainerRef.current.scrollHeight;
                        }
                    }, 50);
                }
            }
        }
    }, [chatPaginationState, selectedChat]);

    // Intersection Observer for loading more messages
    useEffect(() => {
        const observer = new IntersectionObserver(
            (entries) => {
                const [entry] = entries;
                if (entry.isIntersecting && selectedChat && !isLoadingMoreMessages) {
                    const chatId = selectedChat.type === 'friend' ? selectedChat.PeerID : selectedChat.group_id;
                    const chatState = chatPaginationState[chatId];

                    if (chatState && chatState.hasMore) {
                        setIsLoadingMoreMessages(true);

                        // Store current scroll position relative to the container
                        const container = messagesContainerRef.current;
                        const previousScrollHeight = container.scrollHeight;
                        const previousScrollTop = container.scrollTop;

                        setTimeout(() => {
                            loadMoreMessages(chatId);

                            // Restore scroll position after loading more messages
                            setTimeout(() => {
                                const newScrollHeight = container.scrollHeight;
                                const heightDifference = newScrollHeight - previousScrollHeight;
                                container.scrollTop = previousScrollTop + heightDifference;
                                setIsLoadingMoreMessages(false);
                            }, 50);
                        }, 500); // Small delay to show loading state
                    }
                }
            },
            {
                root: messagesContainerRef.current,
                threshold: 1.0,
            }
        );

        if (loadMoreTriggerRef.current) {
            observer.observe(loadMoreTriggerRef.current);
        }

        return () => {
            if (loadMoreTriggerRef.current) {
                observer.unobserve(loadMoreTriggerRef.current);
            }
        };
    }, [selectedChat, chatPaginationState, isLoadingMoreMessages, loadMoreMessages]);

    // Auto-scroll when new messages arrive (only if user is at bottom)
    useEffect(() => {
        if (selectedChat) {
            const chatId = selectedChat.type === 'friend' ? selectedChat.PeerID : selectedChat.group_id;
            const messages = messagesByChat[chatId];
            const chatState = chatPaginationState[chatId];

            if (messages && messages.length > 0) {
                const container = messagesContainerRef.current;
                if (container) {
                    // For initial load, always scroll to bottom
                    if (chatState && chatState.currentPage === 1) {
                        setTimeout(() => {
                            container.scrollTop = container.scrollHeight;
                        }, 100);
                    } else {
                        // For subsequent messages, only scroll if user is near bottom
                        const isNearBottom = container.scrollHeight - container.scrollTop - container.clientHeight < 100;
                        if (isNearBottom) {
                            setTimeout(() => scrollToBottom('smooth'), 50);
                        }
                    }
                }
            }
        }
    }, [messagesByChat, selectedChat, chatPaginationState, scrollToBottom]);

    // FIXED: Remove immediate message addition, rely only on WebSocket echo
    const handleSendMessage = (e) => {
        e.preventDefault();
        if (!message.trim() || !selectedChat) return;

        const messageToSend = message;
        setMessage(''); // Clear input immediately for better UX

        // Send through WebSocket - don't add to state immediately
        // The message will be added when we receive the WebSocket echo
        if (selectedChat.type === 'friend') {
            websocketService.sendMessage(selectedChat.PeerID, messageToSend);
        } else if (selectedChat.type === 'group') {
            websocketService.sendGroupMessage(selectedChat.group_id, messageToSend);
        }

        // Don't add to state here - wait for WebSocket acknowledgment
        // The message will appear when we receive it back through the WebSocket listener
    };

    const toggleDrawer = () => {
        setDrawerOpen(!drawerOpen);
    };

    const drawerWidth = 240;

    // Get messages for selected chat
    const currentChatId = selectedChat?.type === 'friend' ? selectedChat.PeerID : selectedChat?.group_id;
    const messagesToDisplay = messagesByChat[currentChatId] || [];
    const currentChatState = chatPaginationState[currentChatId];

    if (loading) {
        return (
            <Container sx={{display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh'}}>
                <Box sx={{textAlign: 'center'}}>
                    <CircularProgress color="primary" size={60} thickness={4}/>
                    <Typography variant="h6" color="primary" sx={{mt: 2}}>
                        Loading Chat...
                    </Typography>
                </Box>
            </Container>
        );
    }

    return (
        <Box sx={{display: 'flex', height: '100vh'}}>
            {/* Mobile Drawer */}
            {isMobile && (
                <Drawer
                    variant="temporary"
                    open={drawerOpen}
                    onClose={toggleDrawer}
                    sx={{
                        width: drawerWidth,
                        flexShrink: 0,
                        '& .MuiDrawer-paper': {
                            width: drawerWidth,
                            boxSizing: 'border-box',
                            bgcolor: 'background.default',
                        },
                    }}
                >
                    <Box sx={{display: 'flex', justifyContent: 'flex-end', p: 1}}>
                        <IconButton onClick={toggleDrawer}>
                            <CloseIcon/>
                        </IconButton>
                    </Box>
                    <Sidebar
                        refreshTrigger={refreshTrigger}
                        onSelectChat={handleSelectChat}
                        friends={friends}
                        selectedChat={selectedChat}
                        onDisplayNameUpdate={handleDisplayNameUpdate}
                    />
                </Drawer>
            )}

            {/* Desktop Drawer */}
            {!isMobile && (
                <Drawer
                    variant="permanent"
                    open={drawerOpen}
                    sx={{
                        width: drawerWidth,
                        flexShrink: 0,
                        '& .MuiDrawer-paper': {
                            width: drawerWidth,
                            boxSizing: 'border-box',
                            bgcolor: 'background.default',
                            borderRight: '1px solid rgba(233, 30, 99, 0.12)',
                        },
                    }}
                >
                    <Sidebar
                        refreshTrigger={refreshTrigger}
                        onSelectChat={handleSelectChat}
                        friends={friends}
                        selectedChat={selectedChat}
                        onDisplayNameUpdate={handleDisplayNameUpdate}
                    />
                </Drawer>
            )}

            {/* Main Chat Area */}
            <Box sx={{flexGrow: 1, display: 'flex', flexDirection: 'column', height: '100%'}}>
                {/* Header */}
                <AppBar position="static" color="primary" elevation={0}>
                    <Toolbar>
                        {isMobile && (
                            <IconButton
                                color="inherit"
                                aria-label="open drawer"
                                edge="start"
                                onClick={toggleDrawer}
                                sx={{mr: 2}}
                            >
                                <MenuIcon/>
                            </IconButton>
                        )}
                        <Avatar sx={{
                            bgcolor: 'primary.main',
                            ml: -1,
                            mr: 0.5
                        }}>
                            <img src={chatIcon} alt="Chat Icon" style={{width: '60%', height: '60%'}}/>
                        </Avatar>
                        <Typography variant="h6" component="div" sx={{flexGrow: 1}}>
                            {selectedChat ? (
                                selectedChatDisplayName || (
                                    selectedChat.type === 'friend'
                                        ? (selectedChat.display_name || selectedChat.PeerID)
                                        : (selectedChat.name || `Group (${selectedChat.members?.length || 0})`)
                                )
                            ) : 'P2P Chat'}
                        </Typography>
                        {selectedChat && (
                            <Typography variant="body2" sx={{color: 'inherit', opacity: 0.7}}>
                                {selectedChat.type === 'friend'
                                    ? (selectedChat.IsOnline ? 'Online' : 'Offline')
                                    : `${selectedChat.members?.length || 0} members`}
                            </Typography>
                        )}
                    </Toolbar>
                </AppBar>

                {/* Chat Content */}
                {selectedChat ? (
                    <>
                        {/* Messages Area */}
                        <Box
                            ref={messagesContainerRef}
                            sx={{
                                flexGrow: 1,
                                bgcolor: 'background.default',
                                p: 2,
                                overflowY: 'auto',
                                display: 'flex',
                                flexDirection: 'column',
                                scrollBehavior: 'smooth',
                                position: 'relative'
                            }}
                        >
                            {/* Load More Trigger */}
                            {currentChatState?.hasMore && (
                                <Box
                                    ref={loadMoreTriggerRef}
                                    sx={{
                                        display: 'flex',
                                        justifyContent: 'center',
                                        py: 2,
                                        minHeight: '40px'
                                    }}
                                >
                                    {isLoadingMoreMessages ? (
                                        <CircularProgress size={24}/>
                                    ) : (
                                        <Typography variant="caption" color="text.secondary">
                                            Scroll up to load more messages
                                        </Typography>
                                    )}
                                </Box>
                            )}

                            {isLoadingMessages ? (
                                <Box sx={{
                                    display: 'flex',
                                    justifyContent: 'center',
                                    alignItems: 'center',
                                    height: '100%'
                                }}>
                                    <CircularProgress size={40}/>
                                </Box>
                            ) : messagesToDisplay.length === 0 ? (
                                <Box sx={{
                                    display: 'flex',
                                    justifyContent: 'center',
                                    alignItems: 'center',
                                    height: '100%',
                                    color: 'text.secondary'
                                }}>
                                    <Typography variant="body1">
                                        No messages yet. Start the conversation!
                                    </Typography>
                                </Box>
                            ) : (
                                <>
                                    {messagesToDisplay.map((msg, index) => (
                                        <ChatMessage
                                            key={`${msg.chatId}-${msg.Time}-${msg.SenderPeerId}-${index}`}
                                            message={{
                                                sender: msg.SenderPeerId,
                                                text: msg.Message,
                                                timestamp: msg.Time,
                                            }}
                                            currentUser={{peerId: ownPeerId}}
                                        />
                                    ))}
                                    {/* Force scroll to bottom on initial render */}
                                    {currentChatState?.currentPage === 1 && (
                                        <div
                                            ref={(el) => {
                                                if (el && messagesContainerRef.current) {
                                                    // Force scroll to bottom immediately
                                                    setTimeout(() => {
                                                        messagesContainerRef.current.scrollTop = messagesContainerRef.current.scrollHeight;
                                                    }, 0);
                                                }
                                            }}
                                        />
                                    )}
                                </>
                            )}
                            <div ref={messagesEndRef}/>
                        </Box>

                        {/* Message Input */}
                        <Paper
                            component="form"
                            onSubmit={handleSendMessage}
                            sx={{
                                p: 2,
                                display: 'flex',
                                alignItems: 'center',
                                borderTop: '1px solid',
                                borderColor: 'divider',
                                borderRadius: 0,
                            }}
                            elevation={0}
                        >
                            <TextField
                                fullWidth
                                variant="outlined"
                                placeholder="Type your message..."
                                value={message}
                                onChange={(e) => setMessage(e.target.value)}
                                onKeyPress={(e) => {
                                    if (e.key === 'Enter' && !e.shiftKey) {
                                        e.preventDefault();
                                        handleSendMessage(e);
                                    }
                                }}
                                sx={{
                                    '& .MuiOutlinedInput-root': {
                                        borderRadius: 4,
                                    },
                                }}
                            />
                            <IconButton
                                type="submit"
                                color="primary"
                                disabled={!message.trim()}
                                sx={{
                                    ml: 1,
                                    bgcolor: 'primary.main',
                                    color: 'white',
                                    '&:hover': {
                                        bgcolor: 'primary.dark'
                                    },
                                    '&:disabled': {
                                        bgcolor: 'grey.300',
                                        color: 'grey.500'
                                    }
                                }}
                            >
                                <SendIcon/>
                            </IconButton>
                        </Paper>
                    </>
                ) : (
                    <Box sx={{
                        flexGrow: 1,
                        bgcolor: 'background.default',
                        display: 'flex',
                        justifyContent: 'center',
                        alignItems: 'center'
                    }}>
                        <Typography variant="h6" color="text.secondary">
                            Select a chat to start messaging
                        </Typography>
                    </Box>
                )}
            </Box>
        </Box>
    );
}

export default ChatPage;