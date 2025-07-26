import React, {useCallback, useEffect, useRef, useState} from 'react';
import {useNavigate} from 'react-router-dom';
import {
    AppBar,
    Avatar,
    Box,
    CircularProgress,
    Container,
    Drawer,
    IconButton,
    Paper,
    TextField,
    Toolbar,
    Typography,
    useMediaQuery,
    useTheme
} from '@mui/material';
import MenuIcon from '@mui/icons-material/Menu';
import CloseIcon from '@mui/icons-material/Close';
import SendIcon from '@mui/icons-material/Send';
import ChatMessage from './ChatMessage';
import Sidebar from '../sidebar/Sidebar';
import {DAEMON_STATES} from '../utils/constants';
import {checkStatus, getChatMessages, getDisplayNameAPI, getFriends, getGroupChatMessages} from "../../services/api.js";
import chatIcon from '../../../public/icon.svg';
import websocketService from '../../services/websocket';

const getPeerId = () => {
    const peerId = localStorage.getItem('userPeerId') || localStorage.getItem('peerID') || 'mock-peer-id';
    return peerId.trim();
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
    const [message, setMessage] = useState('');
    const [messagesByChat, setMessagesByChat] = useState({});
    const [selectedChat, setSelectedChat] = useState(null);
    const [ownPeerId, setOwnPeerId] = useState('');
    const [friends, setFriends] = useState([]);
    const [isLoadingMessages, setIsLoadingMessages] = useState(false);
    const [isLoadingMoreMessages, setIsLoadingMoreMessages] = useState(false);
    const [selectedChatDisplayName, setSelectedChatDisplayName] = useState('');
    const [chatPaginationState, setChatPaginationState] = useState({});

    const handleDisplayNameUpdate = useCallback((entityId, entityType, newDisplayName) => {
        if (entityType === 'friend') {
            setFriends(prev => prev.map(friend =>
                friend.PeerID === entityId
                    ? {...friend, display_name: newDisplayName || undefined}
                    : friend
            ));
        }

        if (selectedChat) {
            const currentEntityId = selectedChat.type === 'friend' ? selectedChat.PeerID : selectedChat.group_id;
            if (currentEntityId === entityId && selectedChat.type === entityType) {
                if (newDisplayName) {
                    setSelectedChatDisplayName(newDisplayName);
                } else {
                    if (selectedChat.type === 'friend') {
                        setSelectedChatDisplayName(selectedChat.PeerID);
                    } else {
                        setSelectedChatDisplayName(selectedChat.name || `Group (${selectedChat.members?.length || 0})`);
                    }
                }
            }
        }
    }, [selectedChat]);

    const loadSelectedChatDisplayName = useCallback(async (chat) => {
        if (!chat) {
            setSelectedChatDisplayName('');
            return;
        }
        const entityId = chat.type === 'friend' ? chat.PeerID : chat.group_id;
        const entityType = chat.type;
        if (chat.type === 'friend' && chat.display_name) {
            setSelectedChatDisplayName(chat.display_name);
            return;
        }

        try {
            if (chat.type === 'group') {
                setSelectedChatDisplayName(entityId.name);
            } else {
                const displayNameData = await getDisplayNameAPI(entityId, entityType);
                if (displayNameData && displayNameData.display_name) {
                    setSelectedChatDisplayName(displayNameData.display_name);
                } else {
                    if (chat.type === 'friend') {
                        setSelectedChatDisplayName(chat.PeerID);
                    } else {
                        setSelectedChatDisplayName(chat.name || `Group (${chat.members?.length || 0})`);
                    }
                }
            }
        } catch (error) {
            if (chat.type === 'friend') {
                setSelectedChatDisplayName(chat.PeerID);
            } else {
                setSelectedChatDisplayName(chat.name || `Group (${chat.members?.length || 0})`);
            }
        }
    }, []);

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

    const scrollToBottom = useCallback((behavior = 'smooth', force = false) => {
        if (messagesEndRef.current) {
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
        websocketService.connect();
        const removeListener = websocketService.addMessageListener((data) => {
            console.log("WebSocket message received:", data);
            if (!ownPeerId) {
                console.warn("Received WebSocket message before ownPeerId was set. Message:", data);
                return;
            }
            if (data.type === 'DIRECT_MESSAGE' || data.type === 'GROUP_MESSAGE') {
                const {sender_peer_id, message: chatMessageText, target_peer_id, group_id} = data.payload;
                console.log("Processing WebSocket message:", {
                    type: data.type,
                    sender: sender_peer_id,
                    message: chatMessageText,
                    ownPeerId: ownPeerId,
                    isMyMessage: sender_peer_id === ownPeerId
                });
                let chatId;
                if (data.type === 'DIRECT_MESSAGE') {
                    chatId = sender_peer_id.trim() === ownPeerId.trim() ? target_peer_id : sender_peer_id;
                } else {
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
                setMessagesByChat(prev => {
                    const chatMessages = prev[chatId] || [];
                    return {
                        ...prev,
                        [chatId]: [...chatMessages, newMessage]
                    };
                });
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
            return fetchedMsgs.sort((a, b) => new Date(a.Time).getTime() - new Date(b.Time).getTime());
        } catch (error) {
            console.error('Failed to load messages:', error);
            return [];
        }
    }, [ownPeerId]);

    const getLastNMessages = useCallback((messages, n) => {
        return messages.slice(-n);
    }, []);
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

    const handleSelectChat = useCallback(async (chat) => {
        console.log('Selected chat:', chat);
        if (selectedChat?.PeerID === chat.PeerID && selectedChat?.group_id === chat.group_id) {
            return;
        }
        setSelectedChat(chat);
        setMessage('');
        await loadSelectedChatDisplayName(chat);
        const chatId = chat.type === 'friend' ? chat.PeerID : chat.group_id;
        if (chatPaginationState[chatId]) {
            setTimeout(() => scrollToBottom('auto', true), 50);
            return;
        }
        setIsLoadingMessages(true);
        try {
            const allMessages = await loadAllMessages(chat);
            const initialMessages = getLastNMessages(allMessages, MESSAGES_PER_PAGE);
            setChatPaginationState(prev => ({
                ...prev,
                [chatId]: {
                    allMessages: allMessages,
                    displayedMessages: initialMessages,
                    hasMore: allMessages.length > MESSAGES_PER_PAGE,
                    currentPage: 1
                }
            }));
            setMessagesByChat(prev => ({
                ...prev,
                [chatId]: initialMessages,
            }));
            setTimeout(() => scrollToBottom('auto', true), 100);
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

    useEffect(() => {
        if (selectedChat) {
            const chatId = selectedChat.type === 'friend' ? selectedChat.PeerID : selectedChat.group_id;
            const chatState = chatPaginationState[chatId];
            if (chatState) {
                setMessagesByChat(prev => ({
                    ...prev,
                    [chatId]: chatState.displayedMessages
                }));
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

    useEffect(() => {
        const observer = new IntersectionObserver(
            (entries) => {
                const [entry] = entries;
                if (entry.isIntersecting && selectedChat && !isLoadingMoreMessages) {
                    const chatId = selectedChat.type === 'friend' ? selectedChat.PeerID : selectedChat.group_id;
                    const chatState = chatPaginationState[chatId];
                    if (chatState && chatState.hasMore) {
                        setIsLoadingMoreMessages(true);
                        const container = messagesContainerRef.current;
                        const previousScrollHeight = container.scrollHeight;
                        const previousScrollTop = container.scrollTop;
                        setTimeout(() => {
                            loadMoreMessages(chatId);
                            setTimeout(() => {
                                const newScrollHeight = container.scrollHeight;
                                const heightDifference = newScrollHeight - previousScrollHeight;
                                container.scrollTop = previousScrollTop + heightDifference;
                                setIsLoadingMoreMessages(false);
                            }, 50);
                        }, 500);
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

    useEffect(() => {
        if (selectedChat) {
            const chatId = selectedChat.type === 'friend' ? selectedChat.PeerID : selectedChat.group_id;
            const messages = messagesByChat[chatId];
            const chatState = chatPaginationState[chatId];
            if (messages && messages.length > 0) {
                const container = messagesContainerRef.current;
                if (container) {
                    if (chatState && chatState.currentPage === 1) {
                        setTimeout(() => {
                            container.scrollTop = container.scrollHeight;
                        }, 100);
                    } else {
                        const isNearBottom = container.scrollHeight - container.scrollTop - container.clientHeight < 100;
                        if (isNearBottom) {
                            setTimeout(() => scrollToBottom('smooth'), 50);
                        }
                    }
                }
            }
        }
    }, [messagesByChat, selectedChat, chatPaginationState, scrollToBottom]);

    const handleSendMessage = (e) => {
        e.preventDefault();
        if (!message.trim() || !selectedChat) return;

        const messageToSend = message;
        setMessage('');
        if (selectedChat.type === 'friend') {
            websocketService.sendMessage(selectedChat.PeerID, messageToSend);
        } else if (selectedChat.type === 'group') {
            websocketService.sendGroupMessage(selectedChat.group_id, messageToSend);
        }
    };

    const toggleDrawer = () => {
        setDrawerOpen(!drawerOpen);
    };

    const drawerWidth = 240;
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

            <Box sx={{flexGrow: 1, display: 'flex', flexDirection: 'column', height: '100%'}}>
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

                {selectedChat ? (
                    <>
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
                                    {currentChatState?.currentPage === 1 && (
                                        <div
                                            ref={(el) => {
                                                if (el && messagesContainerRef.current) {
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