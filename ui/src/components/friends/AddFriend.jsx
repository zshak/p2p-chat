import React, { useState } from 'react';
import {
    Dialog,
    DialogTitle,
    DialogContent,
    DialogActions,
    TextField,
    Button,
    Alert,
    Box,
    Typography
} from '@mui/material';
import PersonAddIcon from '@mui/icons-material/PersonAdd';

const AddFriend = ({ open, onClose, onFriendRequestSent }) => {
    const [peerId, setPeerId] = useState('');
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState('');
    const [success, setSuccess] = useState('');

    const handleSubmit = async (e) => {
        e.preventDefault();
        if (!peerId.trim()) {
            setError('Peer ID cannot be empty');
            return;
        }

        setLoading(true);
        setError('');
        setSuccess('');

        try {
            await sendFriendRequest(peerId.trim());
            setSuccess('Friend request sent successfully!');
            setPeerId('');
            if (onFriendRequestSent) {
                onFriendRequestSent();
            }
            setTimeout(() => {
                setSuccess('');
                onClose();
            }, 2000);
        } catch (err) {
            setError(err.response?.data || 'Failed to send friends request');
        } finally {
            setLoading(false);
        }
    };

    const handleClose = () => {
        setPeerId('');
        setError('');
        setSuccess('');
        onClose();
    };

    return (
        <Dialog open={open} onClose={handleClose} maxWidth="sm" fullWidth>
            <DialogTitle>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                    <PersonAddIcon color="primary" />
                    Add Friend
                </Box>
            </DialogTitle>
            <DialogContent>
                <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                    Enter the Peer ID of the person you want to add as a friend.
                </Typography>

                {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}
                {success && <Alert severity="success" sx={{ mb: 2 }}>{success}</Alert>}

                <TextField
                    autoFocus
                    margin="dense"
                    label="Peer ID"
                    fullWidth
                    variant="outlined"
                    value={peerId}
                    onChange={(e) => setPeerId(e.target.value)}
                    disabled={loading}
                    placeholder="Enter peer ID..."
                />
            </DialogContent>
            <DialogActions>
                <Button onClick={handleClose} disabled={loading}>
                    Cancel
                </Button>
                <Button
                    onClick={handleSubmit}
                    variant="contained"
                    disabled={loading || !peerId.trim()}
                >
                    {loading ? 'Sending...' : 'Send Request'}
                </Button>
            </DialogActions>
        </Dialog>
    );
};

export default AddFriend;