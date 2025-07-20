import React, { useCallback, useEffect, useState } from 'react';
import {
    Avatar,
    Badge,
    Box,
    Button,
    Divider,
    IconButton,
    List,
    ListItem,
    ListItemText,
    Tooltip,
    Typography,
    Snackbar,
    Alert
} from '@mui/material';
import PersonIcon from '@mui/icons-material/Person';
import SettingsIcon from '@mui/icons-material/Settings';
import LogoutIcon from '@mui/icons-material/Logout';
import PersonAddIcon from '@mui/icons-material/PersonAdd';
import NotificationsIcon from '@mui/icons-material/Notifications';
import GroupIcon from '@mui/icons-material/Group';
import AddBoxIcon from '@mui/icons-material/AddBox';
import ContentCopyIcon from '@mui/icons-material/ContentCopy';
import { useNavigate } from 'react-router-dom';
import { getFriendRequests, getFriends, getGroupChats, checkStatus } from '../../services/api';
import AddFriend from '../friends/AddFriend';
import FriendRequests from '../friends/FriendRequests';
import CreateGroupChat from '../groupchat/CreateGroupChat.jsx';

const Sidebar = ({ refreshTrigger = 0, onSelectChat }) => {
    const navigate = useNavigate();
    const [friends, setFriends] = useState([]);
    const [groupChats, setGroupChats] = useState([]);
    const [friendRequests, setFriendRequests] = useState([]);
    const [currentUser, setCurrentUser] = useState(null);
    const [loading, setLoading] = useState(true);
    const [addFriendOpen, setAddFriendOpen] = useState(false);
    const [friendRequestsOpen, setFriendRequestsOpen] = useState(false);
    const [createGroupChatOpen, setCreateGroupChatOpen] = useState(false);
    const [copySuccess, setCopySuccess] = useState(false);

    const formatPeerId = (peerId) => {
        if (!peerId || peerId.length < 8) return peerId;
        const first2 = peerId.substring(0, 2);
        const last6 = peerId.substring(peerId.length - 6);
        return `${first2}*${last6}`;
    };

    const getDisplayName = (chat) => {
        if (chat.PeerID) {
            return chat.display_name || formatPeerId(chat.PeerID);
        } else if (chat.group_id) {
            return chat.name || `Group (${chat.members?.length || 0})`;
        }
        return 'Unknown Chat';
    };

    const getInitial = (chat) => {
        const displayName = getDisplayName(chat);
        return displayName.charAt(0).toUpperCase();
    };

    const loadChatData = useCallback(async () => {
        try {
            const [friendsResponse, requestsResponse, groupChatsResponse, statusResponse] = await Promise.all([
                getFriends(),
                getFriendRequests(),
                getGroupChats(),
                checkStatus()
            ]);

            const validFriends = (friendsResponse.data || []).filter(friend =>
                friend && friend.PeerID
            );

            setFriends(validFriends);
            setFriendRequests(requestsResponse.data || []);
            setGroupChats(groupChatsResponse.data || []);
            setCurrentUser({
                peer_id: statusResponse.data?.peer_id || null,
                state: statusResponse.data?.state || null
            });
        } catch (error) {
            console.error('Failed to load chat data:', error);
            setFriends([]);
            setFriendRequests([]);
            setGroupChats([]);
            setCurrentUser(null);
        } finally {
            setLoading(false);
        }
    }, []);

    useEffect(() => {
        loadChatData();
    }, [loadChatData]);

    useEffect(() => {
        if (refreshTrigger > 0) {
            loadChatData();
        }
    }, [refreshTrigger, loadChatData]);

    useEffect(() => {
        const interval = setInterval(() => {
            if (!document.hidden) {
                loadChatData();
            }
        }, 10000);

        return () => clearInterval(interval);
    }, [loadChatData]);

    const handleFriendRequestSent = () => {
        loadChatData();
    };

    const handleRequestHandled = (peerId, isAccepted) => {
        setFriendRequests(prev => prev.filter(req => req.PeerID !== peerId));
        if (isAccepted) {
            loadChatData();
        }
    };

    const handleGroupChatCreated = () => {
        loadChatData();
    };

    const handleSelectChat = (chat, type) => {
        if (onSelectChat) {
            onSelectChat({ ...chat, type });
        }
    };

    const handleCopyPeerId = async () => {
        if (!currentUser?.peer_id) return;

        try {
            await navigator.clipboard.writeText(currentUser.peer_id);
            setCopySuccess(true);
        } catch (err) {
            // Fallback for older browsers
            const textArea = document.createElement('textarea');
            textArea.value = currentUser.peer_id;
            document.body.appendChild(textArea);
            textArea.select();
            try {
                document.execCommand('copy');
                setCopySuccess(true);
            } catch (fallbackErr) {
                console.error('Failed to copy peer ID:', fallbackErr);
            }
            document.body.removeChild(textArea);
        }
    };

    const handleCloseCopySnackbar = () => {
        setCopySuccess(false);
    };

    const renderChatSection = (title, items, type, emptyMessage) => (
        <>
            <Typography
                variant="subtitle2"
                color="text.secondary"
                sx={{
                    pl: 1,
                    mb: 1,
                    mt: type === 'friend' ? 2 : 0,
                    fontWeight: 600,
                    textTransform: 'uppercase',
                    letterSpacing: 0.5
                }}
            >
                {title} ({items.length})
            </Typography>

            {loading ? (
                <Typography variant="body2" color="text.secondary" sx={{ pl: 1, py: 1 }}>
                    Loading {type === 'group' ? 'groups' : 'friends'}...
                </Typography>
            ) : items.length === 0 ? (
                <Typography variant="body2" color="text.secondary" sx={{ pl: 1, py: 1 }}>
                    {emptyMessage}
                </Typography>
            ) : (
                <List sx={{ py: 0 }}>
                    {items.map((item) => (
                        <ListItem
                            button
                            key={type === 'group' ? item.group_id : item.PeerID}
                            onClick={() => handleSelectChat(item, type)}
                            sx={{
                                borderRadius: 1,
                                mb: 0.5,
                                mx: 1,
                                transition: 'all 0.2s ease-in-out',
                                '&:hover': {
                                    bgcolor: 'primary.light',
                                    '& .MuiTypography-root': {
                                        color: 'primary.contrastText',
                                    },
                                },
                            }}
                        >
                            {type === 'group' ? (
                                <Avatar sx={{ bgcolor: 'info.main', mr: 2, width: 40, height: 40 }}>
                                    <GroupIcon />
                                </Avatar>
                            ) : (
                                <Badge
                                    color={item.IsOnline ? 'success' : 'error'}
                                    variant="dot"
                                    anchorOrigin={{
                                        vertical: 'bottom',
                                        horizontal: 'right',
                                    }}
                                    overlap="circular"
                                    sx={{ mr: 2 }}
                                >
                                    <Avatar sx={{ bgcolor: 'secondary.light', width: 40, height: 40 }}>
                                        {getInitial(item)}
                                    </Avatar>
                                </Badge>
                            )}

                            <ListItemText
                                primary={getDisplayName(item)}
                                secondary={
                                    type === 'group'
                                        ? `${item.members?.length || 0} members`
                                        : formatPeerId(item.PeerID)
                                }
                                primaryTypographyProps={{
                                    noWrap: true,
                                    fontSize: 14,
                                    fontWeight: 500
                                }}
                                secondaryTypographyProps={{
                                    noWrap: true,
                                    fontSize: 12,
                                    color: 'text.secondary'
                                }}
                            />
                        </ListItem>
                    ))}
                </List>
            )}
        </>
    );

    return (
        <>
            {/* User Profile Section */}
            <Box
                sx={{
                    display: 'flex',
                    flexDirection: 'column',
                    alignItems: 'center',
                    p: 2,
                    bgcolor: 'background.paper'
                }}
            >
                <Avatar
                    sx={{
                        width: 64,
                        height: 64,
                        bgcolor: 'primary.main',
                        mb: 1
                    }}
                >
                    <PersonIcon fontSize="large" />
                </Avatar>

                {/* Peer ID Display with Copy Button */}
                <Box sx={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 1,
                    mb: 1,
                    maxWidth: '100%'
                }}>
                    <Typography
                        variant="body2"
                        color="primary.dark"
                        sx={{
                            fontFamily: 'monospace',
                            fontWeight: 600,
                            fontSize: '0.75rem',
                            wordBreak: 'break-all',
                            textAlign: 'center',
                            maxWidth: '150px'
                        }}
                    >
                        {currentUser?.peer_id ? formatPeerId(currentUser.peer_id) : 'Loading...'}
                    </Typography>

                    {currentUser?.peer_id && (
                        <Tooltip title="Copy full Peer ID" arrow>
                            <IconButton
                                size="small"
                                onClick={handleCopyPeerId}
                                sx={{
                                    width: 24,
                                    height: 24,
                                    bgcolor: 'primary.light',
                                    color: 'primary.contrastText',
                                    '&:hover': {
                                        bgcolor: 'primary.main'
                                    }
                                }}
                            >
                                <ContentCopyIcon fontSize="inherit" />
                            </IconButton>
                        </Tooltip>
                    )}
                </Box>

                <Typography variant="body2" color="text.secondary">
                    Online
                </Typography>
            </Box>

            <Divider />

            {/* Action Buttons */}
            <Box sx={{ p: 2, display: 'flex', gap: 1, bgcolor: 'background.paper' }}>
                <Tooltip title="Add Friend" arrow>
                    <IconButton
                        color="primary"
                        onClick={() => setAddFriendOpen(true)}
                        sx={{
                            flex: 1,
                            '&:hover': {
                                bgcolor: 'primary.light',
                                color: 'primary.contrastText'
                            }
                        }}
                    >
                        <PersonAddIcon />
                    </IconButton>
                </Tooltip>

                <Tooltip title="Friend Requests" arrow>
                    <IconButton
                        color="primary"
                        onClick={() => setFriendRequestsOpen(true)}
                        sx={{
                            flex: 1,
                            position: 'relative',
                            '&:hover': {
                                bgcolor: 'primary.light',
                                color: 'primary.contrastText'
                            }
                        }}
                    >
                        <NotificationsIcon />
                        {friendRequests.length > 0 && (
                            <Badge
                                badgeContent={friendRequests.length}
                                color="error"
                                sx={{
                                    position: 'absolute',
                                    top: 8,
                                    right: 8
                                }}
                            />
                        )}
                    </IconButton>
                </Tooltip>

                <Tooltip title="Create Group Chat" arrow>
                    <IconButton
                        color="primary"
                        onClick={() => setCreateGroupChatOpen(true)}
                        sx={{
                            flex: 1,
                            '&:hover': {
                                bgcolor: 'primary.light',
                                color: 'primary.contrastText'
                            }
                        }}
                    >
                        <AddBoxIcon />
                    </IconButton>
                </Tooltip>
            </Box>

            <Divider />

            {/* Chat Lists */}
            <Box
                sx={{
                    flexGrow: 1,
                    overflowY: 'auto',
                    bgcolor: 'background.default',
                    p: 1
                }}
            >
                {renderChatSection(
                    'Group Chats',
                    groupChats,
                    'group',
                    'No group chats yet.'
                )}

                {renderChatSection(
                    'Friends',
                    friends,
                    'friend',
                    'No friends yet. Add some friends to start chatting!'
                )}
            </Box>

            {/* Copy Success Snackbar */}
            <Snackbar
                open={copySuccess}
                autoHideDuration={3000}
                onClose={handleCloseCopySnackbar}
                anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}
            >
                <Alert
                    onClose={handleCloseCopySnackbar}
                    severity="success"
                    variant="filled"
                    sx={{ width: '100%' }}
                >
                    Peer ID copied to clipboard!
                </Alert>
            </Snackbar>

            {/* Modals */}
            <AddFriend
                open={addFriendOpen}
                onClose={() => setAddFriendOpen(false)}
                onFriendRequestSent={handleFriendRequestSent}
            />

            <FriendRequests
                open={friendRequestsOpen}
                onClose={() => setFriendRequestsOpen(false)}
                friendRequests={friendRequests}
                onRequestHandled={handleRequestHandled}
            />

            <CreateGroupChat
                open={createGroupChatOpen}
                onClose={() => setCreateGroupChatOpen(false)}
                onCreateGroupChat={handleGroupChatCreated}
                friends={friends}
            />
        </>
    );
};

export default Sidebar;