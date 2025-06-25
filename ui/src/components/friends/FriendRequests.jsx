import React, { useState } from 'react';
import {
    Dialog,
    DialogTitle,
    DialogContent,
    List,
    ListItem,
    ListItemText,
    ListItemSecondaryAction,
    Button,
    Typography,
    Box,
    Alert,
    Avatar,
    Chip
} from '@mui/material';
import PersonIcon from '@mui/icons-material/Person';
import CheckIcon from '@mui/icons-material/Check';
import CloseIcon from '@mui/icons-material/Close';
import { respondToFriendRequest } from '../../services/api';

const FriendRequests = ({ open, onClose, friendRequests, onRequestHandled }) => {
    const [loading, setLoading] = useState({});
    const [error, setError] = useState('');

    const handleResponse = async (peerId, isAccepted) => {
        setLoading(prev => ({ ...prev, [peerId]: true }));
        setError('');

        try {
            await respondToFriendRequest(peerId, isAccepted);
            if (onRequestHandled) {
                onRequestHandled(peerId, isAccepted);
            }
        } catch (err) {
            setError(err.response?.data || 'Failed to respond to friends request');
        } finally {
            setLoading(prev => ({ ...prev, [peerId]: false }));
        }
    };

    return (
        <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
            <DialogTitle>Friend Requests</DialogTitle>
            <DialogContent>
                {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}

                {friendRequests.length === 0 ? (
                    <Typography variant="body2" color="text.secondary" align="center" sx={{ py: 3 }}>
                        No pending friend requests
                    </Typography>
                ) : (
                    <List>
                        {friendRequests.map((request) => (
                            <ListItem key={request.peerId} divider>
                                <Avatar sx={{ bgcolor: 'secondary.main', mr: 2 }}>
                                    <PersonIcon />
                                </Avatar>
                                <ListItemText
                                    primary={request.displayName || request.peerId}
                                    secondary={`Peer ID: ${request.peerId}`}
                                />
                                <ListItemSecondaryAction>
                                    <Box sx={{ display: 'flex', gap: 1 }}>
                                        <Button
                                            size="small"
                                            variant="contained"
                                            color="success"
                                            startIcon={<CheckIcon />}
                                            onClick={() => handleResponse(request.peerId, true)}
                                            disabled={loading[request.peerId]}
                                        >
                                            Accept
                                        </Button>
                                        <Button
                                            size="small"
                                            variant="outlined"
                                            color="error"
                                            startIcon={<CloseIcon />}
                                            onClick={() => handleResponse(request.peerId, false)}
                                            disabled={loading[request.peerId]}
                                        >
                                            Decline
                                        </Button>
                                    </Box>
                                </ListItemSecondaryAction>
                            </ListItem>
                        ))}
                    </List>
                )}
            </DialogContent>
        </Dialog>
    );
};

export default FriendRequests;