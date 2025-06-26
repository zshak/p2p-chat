import React, {useEffect, useState} from 'react';
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
import {useNavigate} from 'react-router-dom';
import {getFriendRequests, getFriends} from '../../services/api';
import AddFriend from '../friends/AddFriend';
import FriendRequests from '../friends/FriendRequests';

const Sidebar = () => {
    const navigate = useNavigate();
    const [friends, setFriends] = useState([]);
    const [friendRequests, setFriendRequests] = useState([]);
    const [loading, setLoading] = useState(true);
    const [addFriendOpen, setAddFriendOpen] = useState(false);
    const [friendRequestsOpen, setFriendRequestsOpen] = useState(false);
    const formatPeerId = (peerId) => {
        if (!peerId || peerId.length < 8) return peerId;
        const first2 = peerId.substring(0, 2);
        const last6 = peerId.substring(peerId.length - 6);
        return `${first2}*${last6}`;
    };

    useEffect(() => {
        loadFriendsData();
    }, []);

    const loadFriendsData = async () => {
        try {
            // Load both friends and friend requests in parallel
            const [friendsResponse, requestsResponse] = await Promise.all([
                getFriends(),
                getFriendRequests()
            ]);

            setFriends(friendsResponse.data || []);
            setFriendRequests(requestsResponse.data || []);
        } catch (error) {
            console.error('Failed to load friends data:', error);
            // Set empty arrays on error to prevent UI issues
            setFriends([]);
            setFriendRequests([]);
        } finally {
            setLoading(false);
        }
    };

    const handleFriendRequestSent = () => {
        loadFriendsData(); // Refresh both friends and requests
    };

    const handleRequestHandled = (peerId, isAccepted) => {
        // Remove the request from pending requests
        setFriendRequests(prev => prev.filter(req => req.PeerID !== peerId));

        // If accepted, refresh friends list to include the new friend
        if (isAccepted) {
            loadFriendsData();
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
                    Your Name
                </Typography>
                <Typography variant="body2" color="text.secondary">
                    Online
                </Typography>
            </Box>

            <Divider/>

            {/* Friend Management Buttons */}
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
            </Box>

            <Divider/>

            <Box sx={{p: 2, flexGrow: 1}}>
                <Typography variant="subtitle2" color="text.secondary" sx={{pl: 1, mb: 1}}>
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
                                key={friend.peer_id}
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
                                    color={friend.isOnline ? 'success' : 'error'}
                                    variant="dot"
                                    anchorOrigin={{
                                        vertical: 'bottom',
                                        horizontal: 'right',
                                    }}
                                    overlap="circular"
                                    sx={{mr: 2}}
                                >
                                    <Avatar sx={{bgcolor: 'secondary.light'}}>
                                        {(friend.display_name || friend.peer_id).charAt(0).toUpperCase()}
                                    </Avatar>
                                </Badge>
                                <ListItemText
                                    primary={friend.display_name || formatPeerId(friend.peer_id)}
                                    secondary={formatPeerId(friend.peer_id)}
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
                    Settings
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

            {/* Add Friend Dialog */}
            <AddFriend
                open={addFriendOpen}
                onClose={() => setAddFriendOpen(false)}
                onFriendRequestSent={handleFriendRequestSent}
            />

            {/* Friend Requests Dialog */}
            <FriendRequests
                open={friendRequestsOpen}
                onClose={() => setFriendRequestsOpen(false)}
                friendRequests={friendRequests}
                onRequestHandled={handleRequestHandled}
            />
        </>
    );
};

export default Sidebar;