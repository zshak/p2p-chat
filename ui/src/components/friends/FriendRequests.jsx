import React from 'react';
import {
    Avatar,
    Box,
    Button,
    Dialog,
    DialogActions,
    DialogContent,
    DialogTitle,
    Divider,
    IconButton,
    List,
    ListItem,
    ListItemSecondaryAction,
    ListItemText,
    Typography
} from '@mui/material';
import CheckIcon from '@mui/icons-material/Check';
import CloseIcon from '@mui/icons-material/Close';
import PersonIcon from '@mui/icons-material/Person';
import {respondToFriendRequest} from '../../services/api';

const FriendRequests = ({open, onClose, friendRequests, onRequestHandled}) => {
    const pendingRequests = friendRequests.filter(request => request.Status === 2);
    const sentRequests = friendRequests.filter(request => request.Status === 1);

    const handleAccept = async (peerId) => {
        try {
            await respondToFriendRequest(peerId, true);
            onRequestHandled(peerId, true);
        } catch (error) {
            console.error('Failed to accept friend request:', error);
            // You might want to show an error message to the user
        }
    };

    const handleReject = async (peerId) => {
        try {
            await respondToFriendRequest(peerId, false);
            onRequestHandled(peerId, false);
        } catch (error) {
            console.error('Failed to reject friend request:', error);
            // You might want to show an error message to the user
        }
    };

    const formatPeerId = (peerId) => {
        if (!peerId || peerId.length < 8) return peerId;
        const first2 = peerId.substring(0, 2);
        const last6 = peerId.substring(peerId.length - 6);
        return `${first2}*${last6}`;
    };

    const formatDate = (timestamp) => {
        if (!timestamp || timestamp === "0001-01-01T00:00:00Z") return '';
        const date = new Date(timestamp);
        return date.toLocaleDateString() + ' ' + date.toLocaleTimeString([], {
            hour: '2-digit',
            minute: '2-digit'
        });
    };

    const renderRequestItem = (request, index, isLastItem, isPending) => (
        <React.Fragment key={request.PeerID}>
            <ListItem sx={{py: 2}}>
                <Avatar sx={{bgcolor: 'primary.main', mr: 2}}>
                    {request.PeerID.charAt(0).toUpperCase()}
                </Avatar>
                <ListItemText
                    primary={request.DisplayName || formatPeerId(request.PeerID)}
                    secondary={
                        <span>
                            <Typography variant="caption" color="text.secondary" component="span">
                                {formatPeerId(request.PeerID)}
                            </Typography>
                            {request.RequestedAt && request.RequestedAt !== "0001-01-01T00:00:00Z" && (
                                <Typography variant="caption" color="text.secondary"
                                            component="span" sx={{display: 'block'}}>
                                    {isPending ? "Received: " : "Sent: "}{formatDate(request.RequestedAt)}
                                </Typography>
                            )}
                        </span>
                    }
                />
                {isPending && (
                    <ListItemSecondaryAction>
                        <Box sx={{display: 'flex', gap: 1}}>
                            <IconButton
                                color="success"
                                onClick={() => handleAccept(request.PeerID)}
                                size="small"
                                sx={{
                                    bgcolor: 'success.light',
                                    '&:hover': {bgcolor: 'success.main'}
                                }}
                            >
                                <CheckIcon/>
                            </IconButton>
                            <IconButton
                                color="error"
                                onClick={() => handleReject(request.PeerID)}
                                size="small"
                                sx={{
                                    bgcolor: 'error.light',
                                    '&:hover': {bgcolor: 'error.main'}
                                }}
                            >
                                <CloseIcon/>
                            </IconButton>
                        </Box>
                    </ListItemSecondaryAction>
                )}
            </ListItem>
            {!isLastItem && <Divider/>}
        </React.Fragment>
    );

    return (
        <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
            <DialogTitle>
                <Box sx={{display: 'flex', alignItems: 'center', gap: 1}}>
                    <PersonIcon color="primary"/>
                    Friend Requests
                    {(pendingRequests.length + sentRequests.length) > 0 && (
                        <Typography variant="caption" color="text.secondary">
                            ({pendingRequests.length + sentRequests.length})
                        </Typography>
                    )}
                </Box>
            </DialogTitle>

            <DialogContent sx={{p: 0}}>
                {pendingRequests.length === 0 && sentRequests.length === 0 ? (
                    <Box sx={{p: 3, textAlign: 'center'}}>
                        <Typography variant="body2" color="text.secondary">
                            No pending friend requests
                        </Typography>
                    </Box>
                ) : (
                    <>
                        {pendingRequests.length > 0 && (
                            <>
                                <Typography variant="subtitle2" sx={{p: 2, bgcolor: 'background.paper'}}>
                                    Pending Approval ({pendingRequests.length})
                                </Typography>
                                <List>
                                    {pendingRequests.map((request, index) => (
                                        renderRequestItem(
                                            request,
                                            index,
                                            index === pendingRequests.length - 1,
                                            true
                                        )
                                    ))}
                                </List>
                            </>
                        )}

                        {sentRequests.length > 0 && (
                            <>
                                {pendingRequests.length > 0 && <Divider />}
                                <Typography variant="subtitle2" sx={{p: 2, bgcolor: 'background.paper'}}>
                                    Sent Requests ({sentRequests.length})
                                </Typography>
                                <List>
                                    {sentRequests.map((request, index) => (
                                        renderRequestItem(
                                            request,
                                            index,
                                            index === sentRequests.length - 1,
                                            false
                                        )
                                    ))}
                                </List>
                            </>
                        )}
                    </>
                )}
            </DialogContent>

            <DialogActions>
                <Button onClick={onClose} color="primary">
                    Close
                </Button>
            </DialogActions>
        </Dialog>
    );
};

export default FriendRequests;