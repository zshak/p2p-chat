// src/components/sidebar/Sidebar.jsx
import React, {useCallback, useEffect, useState} from 'react';
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
    Typography
} from '@mui/material';
import PersonIcon from '@mui/icons-material/Person';
import SettingsIcon from '@mui/icons-material/Settings';
import LogoutIcon from '@mui/icons-material/Logout';
import PersonAddIcon from '@mui/icons-material/PersonAdd';
import NotificationsIcon from '@mui/icons-material/Notifications';
import GroupIcon from '@mui/icons-material/Group'; // Import GroupIcon
import AddBoxIcon from '@mui/icons-material/AddBox'; // Import AddBoxIcon
import {useNavigate} from 'react-router-dom';
import {getFriendRequests, getFriends, getGroupChats} from '../../services/api'; // Import getGroupChats
import AddFriend from '../friends/AddFriend';
import FriendRequests from '../friends/FriendRequests';
import CreateGroupChat from '../groupchat/CreateGroupChat.jsx'; // Import CreateGroupChat

const Sidebar = ({refreshTrigger = 0, onSelectChat}) => { // Add onSelectChat prop
    const navigate = useNavigate();
    const [friends, setFriends] = useState([]);
    const [groupChats, setGroupChats] = useState([]); // State for group chats
    const [friendRequests, setFriendRequests] = useState([]);
    const [loading, setLoading] = useState(true);
    const [addFriendOpen, setAddFriendOpen] = useState(false);
    const [friendRequestsOpen, setFriendRequestsOpen] = useState(false);
    const [createGroupChatOpen, setCreateGroupChatOpen] = useState(false); // State for create group modal


    const formatPeerId = (peerId) => {
        if (!peerId || peerId.length < 8) return peerId;
        const first2 = peerId.substring(0, 2);
        const last6 = peerId.substring(peerId.length - 6);
        return `${first2}*${last6}`;
    };

    const getDisplayName = (chat) => {
        if (chat.PeerID) { // It's a friend
            return chat.display_name || formatPeerId(chat.PeerID);
        } else if (chat.group_id) { // It's a group chat
            return chat.name || `Group (${chat.members.length})`; // Use group name if available, otherwise a default
        }
        return 'Unknown Chat';
    };


    const getInitial = (chat) => {
        const displayName = getDisplayName(chat);
        return displayName.charAt(0).toUpperCase();
    };

    const loadChatData = useCallback(async () => { // Rename to loadChatData
        try {
            const [friendsResponse, requestsResponse, groupChatsResponse] = await Promise.all([ // Fetch group chats
                getFriends(),
                getFriendRequests(),
                getGroupChats()
            ]);

            const validFriends = (friendsResponse.data || []).filter(friend =>
                friend && friend.PeerID
            );

            setFriends(validFriends);
            setFriendRequests(requestsResponse.data || []);
            setGroupChats(groupChatsResponse.data || []); // Set group chats


        } catch (error) {
            console.error('Failed to load chat data:', error);
            setFriends([]);
            setFriendRequests([]);
            setGroupChats([]);
        } finally {
            setLoading(false);
        }
    }, []);

    // Initial load
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
        loadChatData(); // Refresh the list after creating a group
    };

    // Handle selecting either a friend or a group chat
    const handleSelectChat = (chat, type) => {
        if (onSelectChat) {
            onSelectChat({ ...chat, type }); // Pass the chat object and type ('friend' or 'group')
        }
    };


    return (
        <>
            <Box
                sx={{
                    display: 'flex',
                    flexDirection: 'column',
                    alignItems: 'center',
                    p: 2
                }}
            >
                <Avatar
                    sx={{
                        width: 80,
                        height: 80,
                        bgcolor: 'primary.main',
                        mb: 1
                    }}
                >
                    <PersonIcon fontSize="large"/>
                </Avatar>
                <Typography variant="h6" color="primary.dark">
                    Your Name {/* TODO: Replace with actual user name */}
                </Typography>
                <Typography variant="body2" color="text.secondary">
                    Online {/* TODO: Replace with actual online status */}
                </Typography>
            </Box>

            <Divider/>

            <Box sx={{p: 2, display: 'flex', gap: 1}}>
                <Tooltip title="Add Friend">
                    <IconButton
                        color="primary"
                        onClick={() => setAddFriendOpen(true)}
                        sx={{flex: 1}}
                    >
                        <PersonAddIcon/>
                    </IconButton>
                </Tooltip>
                <Tooltip title="Friend Requests">
                    <IconButton
                        color="primary"
                        onClick={() => setFriendRequestsOpen(true)}
                        sx={{flex: 1, position: 'relative'}}
                    >
                        <NotificationsIcon/>
                        {friendRequests.length > 0 && (
                            <Badge
                                badgeContent={friendRequests.length}
                                color="error"
                                sx={{
                                    position: 'absolute',
                                    top: 5,
                                    right: 5
                                }}
                            />
                        )}
                    </IconButton>
                </Tooltip>
                <Tooltip title="Create Group Chat"> {/* Add Create Group Chat button */}
                    <IconButton
                        color="primary"
                        onClick={() => setCreateGroupChatOpen(true)}
                        sx={{flex: 1}}
                    >
                        <AddBoxIcon/>
                    </IconButton>
                </Tooltip>
            </Box>

            <Divider/>

            <Box sx={{p: 2, flexGrow: 1, overflowY: 'auto'}}> {/* Make this section scrollable */}
                <Typography variant="subtitle2" color="text.secondary" sx={{pl: 1, mb: 1}}>
                    GROUP CHATS ({groupChats.length}) {/* Group Chats Section */}
                </Typography>
                {loading ? (
                    <Typography variant="body2" color="text.secondary" sx={{pl: 1}}>
                        Loading groups...
                    </Typography>
                ) : groupChats.length === 0 ? (
                    <Typography variant="body2" color="text.secondary" sx={{pl: 1}}>
                        No group chats yet.
                    </Typography>
                ) : (
                    <List>
                        {groupChats.map((groupChat) => (
                            <ListItem
                                button
                                key={groupChat.group_id}
                                onClick={() => handleSelectChat(groupChat, 'group')} // Select group chat
                                sx={{
                                    borderRadius: 1,
                                    mb: 0.5,
                                    '&:hover': {
                                        bgcolor: 'primary.light',
                                        '& .MuiTypography-root': {
                                            color: 'primary.contrastText',
                                        },
                                    },
                                }}
                            >
                                <Avatar sx={{bgcolor: 'info.main', mr: 2}}> {/* Use a different color for group icons */}
                                    <GroupIcon/>
                                </Avatar>
                                <ListItemText
                                    primary={getDisplayName(groupChat)}
                                    secondary={`${groupChat.members.length} members`}
                                    primaryTypographyProps={{
                                        noWrap: true,
                                        fontSize: 14,
                                        fontWeight: 500
                                    }}
                                    secondaryTypographyProps={{
                                        noWrap: true,
                                        fontSize: 12
                                    }}
                                />
                            </ListItem>
                        ))}
                    </List>
                )}

                <Typography variant="subtitle2" color="text.secondary" sx={{pl: 1, mb: 1, mt: 2}}> {/* Friends Section */}
                    FRIENDS ({friends.length})
                </Typography>

                {loading ? (
                    <Typography variant="body2" color="text.secondary" sx={{pl: 1}}>
                        Loading friends...
                    </Typography>
                ) : friends.length === 0 ? (
                    <Typography variant="body2" color="text.secondary" sx={{pl: 1}}>
                        No friends yet. Add some friends to start chatting!
                    </Typography>
                ) : (
                    <List>
                        {friends.map((friend) => (
                            <ListItem
                                button
                                key={friend.PeerID}
                                onClick={() => handleSelectChat(friend, 'friend')} // Select friend
                                sx={{
                                    borderRadius: 1,
                                    mb: 0.5,
                                    '&:hover': {
                                        bgcolor: 'primary.light',
                                        '& .MuiTypography-root': {
                                            color: 'primary.contrastText',
                                        },
                                    },
                                }}
                            >
                                <Badge
                                    color={friend.IsOnline ? 'success' : 'error'}
                                    variant="dot"
                                    anchorOrigin={{
                                        vertical: 'bottom',
                                        horizontal: 'right',
                                    }}
                                    overlap="circular"
                                    sx={{mr: 2}}
                                >
                                    <Avatar sx={{bgcolor: 'secondary.light'}}>
                                        {getInitial(friend)}
                                    </Avatar>
                                </Badge>
                                <ListItemText
                                    primary={getDisplayName(friend)}
                                    secondary={formatPeerId(friend.PeerID)}
                                    primaryTypographyProps={{
                                        noWrap: true,
                                        fontSize: 14,
                                        fontWeight: 500
                                    }}
                                    secondaryTypographyProps={{
                                        noWrap: true,
                                        fontSize: 12
                                    }}
                                />
                            </ListItem>
                        ))}
                    </List>
                )}
            </Box>

            <Box sx={{p: 2}}>
                <Button
                    fullWidth
                    variant="outlined"
                    color="primary"
                    startIcon={<SettingsIcon/>}
                    sx={{mb: 1}}
                >
                    Settings {/* TODO: Implement Settings Page */}
                </Button>
                <Button
                    fullWidth
                    variant="contained"
                    color="error"
                    startIcon={<LogoutIcon/>}
                    onClick={() => navigate('/login')}
                >
                    Logout
                </Button>
            </Box>

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

            <CreateGroupChat // Add CreateGroupChat modal
                open={createGroupChatOpen}
                onClose={() => setCreateGroupChatOpen(false)}
                onCreateGroupChat={handleGroupChatCreated}
                friends={friends} // Pass friends list to select members
            />
        </>
    );
};

export default Sidebar;