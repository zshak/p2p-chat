import { useState, useEffect, useRef } from 'react';
import { useNavigate } from 'react-router-dom';
import {
    Box,
    Typography,
    AppBar,
    Toolbar,
    Drawer,
    IconButton,
    useTheme,
    useMediaQuery,
    CircularProgress,
    Container, Avatar
} from '@mui/material';
import MenuIcon from '@mui/icons-material/Menu';
import CloseIcon from '@mui/icons-material/Close';
import ChatMessage from './ChatMessage';
import MessageInput from './MessageInput';
import Sidebar from '../sidebar/Sidebar';
import { SAMPLE_MESSAGES, DAEMON_STATES } from '../utils/constants'; // Remove ACTIVE_USERS import
import {checkStatus} from "../../services/api.js";
import chatIcon from '../../../public/icon.svg';

function ChatPage() {
    const navigate = useNavigate();
    const theme = useTheme();
    const isMobile = useMediaQuery(theme.breakpoints.down('md'));
    const [status, setStatus] = useState(null);
    const [loading, setLoading] = useState(true);
    const [messages, setMessages] = useState(SAMPLE_MESSAGES);
    const [newMessage, setNewMessage] = useState('');
    const [drawerOpen, setDrawerOpen] = useState(!isMobile);
    const messagesEndRef = useRef(null);

    useEffect(() => {
        const verifyStatus = async () => {
            try {
                const response = await checkStatus();
                setStatus(response.data.state);

                if (response.data.state !== DAEMON_STATES.RUNNING) {
                    navigate('/login');
                }
                setLoading(false);
            } catch (err) {
                navigate('/login');
            }
        };

        verifyStatus();
    }, [navigate]);

    useEffect(() => {
        messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
    }, [messages]);

    const handleSendMessage = (e) => {
        e.preventDefault();
        if (newMessage.trim() === '') return;

        const newMsg = {
            id: messages.length + 1,
            sender: 'me',
            text: newMessage,
            timestamp: Date.now()
        };

        setMessages([...messages, newMsg]);
        setNewMessage('');
    };

    const toggleDrawer = () => {
        setDrawerOpen(!drawerOpen);
    };

    const drawerWidth = 240;

    if (loading) {
        return (
            <Container sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh' }}>
                <Box sx={{ textAlign: 'center' }}>
                    <CircularProgress color="primary" size={60} thickness={4} />
                    <Typography variant="h6" color="primary" sx={{ mt: 2 }}>
                        Loading Chat...
                    </Typography>
                </Box>
            </Container>
        );
    }

    return (
        <Box sx={{ display: 'flex', height: '100vh' }}>
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
                    <Box sx={{ display: 'flex', justifyContent: 'flex-end', p: 1 }}>
                        <IconButton onClick={toggleDrawer}>
                            <CloseIcon />
                        </IconButton>
                    </Box>
                    <Sidebar />
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
                    <Sidebar />
                </Drawer>
            )}

            <Box sx={{ flexGrow: 1, display: 'flex', flexDirection: 'column', height: '100%' }}>
                <AppBar position="static" color="primary" elevation={0}>
                    <Toolbar>
                        {isMobile && (
                            <IconButton
                                color="inherit"
                                aria-label="open drawer"
                                edge="start"
                                onClick={toggleDrawer}
                                sx={{ mr: 2 }}
                            >
                                <MenuIcon />
                            </IconButton>
                        )}
                        <Avatar sx={{
                            bgcolor: 'primary.main',
                            ml: -1,
                            mr: 0.5
                        }}>
                            <img src={chatIcon} alt="Chat Icon" style={{ width: '60%', height: '60%' }} />
                        </Avatar>
                        <Typography variant="h6" component="div" sx={{ flexGrow: 1 }}>
                            P2P Chat
                        </Typography>
                    </Toolbar>
                </AppBar>

                <Box
                    sx={{
                        flexGrow: 1,
                        bgcolor: 'background.default',
                        p: 2,
                        overflowY: 'auto',
                        display: 'flex',
                        flexDirection: 'column',
                    }}
                >
                    {messages.map((message) => (
                        <ChatMessage
                            key={message.id}
                            message={message}
                        />
                    ))}
                    <div ref={messagesEndRef} />
                </Box>

                <MessageInput
                    newMessage={newMessage}
                    setNewMessage={setNewMessage}
                    handleSendMessage={handleSendMessage}
                />
            </Box>
        </Box>
    );
}

export default ChatPage;