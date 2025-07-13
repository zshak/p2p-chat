// src/components/groupchat/CreateGroupChat.jsx
import React, { useState } from 'react';
import {
    Button,
    Dialog,
    DialogTitle,
    DialogContent,
    DialogActions,
    TextField,
    FormControl,
    InputLabel,
    Select,
    MenuItem,
    Chip,
    Box,
    OutlinedInput,
    Typography
} from '@mui/material';
import { createGroupChat } from '../../services/api';
import {getPeerId} from "../utils/userStore";

const CreateGroupChat = ({ open, onClose, onCreateGroupChat, friends }) => {
    const [groupName, setGroupName] = useState('');
    const [selectedMembers, setSelectedMembers] = useState([]);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState('');
    const ownPeerId = getPeerId(); // Get the current user's peer ID

    const handleCreate = async () => {
        if (!groupName.trim() || selectedMembers.length === 0) {
            setError('Group name and members are required.');
            return;
        }

        setLoading(true);
        setError('');

        // Include the current user's peer ID in the members list
        const allMembers = [...selectedMembers];

        try {
            await createGroupChat(allMembers, groupName);
            // Optionally, fetch the updated group chats list after creation
            if (onCreateGroupChat) {
                onCreateGroupChat();
            }
            onClose();
            setGroupName(''); // Clear form
            setSelectedMembers([]); // Clear form
        } catch (err) {
            console.error('Failed to create group chat:', err);
            setError('Failed to create group chat. Please try again.');
        } finally {
            setLoading(false);
        }
    };

    const handleMemberSelect = (event) => {
        const {
            target: { value },
        } = event;
        setSelectedMembers(
            // On autofill we get a stringified value.
            typeof value === 'string' ? value.split(',') : value,
        );
    };

    // Filter out the current user from the friends list for selection
    const availableFriends = friends.filter(friend => friend.PeerID !== ownPeerId);


    return (
        <Dialog open={open} onClose={onClose} fullWidth maxWidth="sm">
            <DialogTitle>Create New Group Chat</DialogTitle>
            <DialogContent>
                {error && (
                    <Typography color="error" variant="body2" sx={{ mb: 2 }}>
                        {error}
                    </Typography>
                )}
                <TextField
                    autoFocus
                    margin="dense"
                    label="Group Name"
                    type="text"
                    fullWidth
                    variant="outlined"
                    value={groupName}
                    onChange={(e) => setGroupName(e.target.value)}
                    sx={{ mb: 2 }}
                />
                <FormControl fullWidth>
                    <InputLabel id="select-members-label">Select Members</InputLabel>
                    <Select
                        labelId="select-members-label"
                        multiple
                        value={selectedMembers}
                        onChange={handleMemberSelect}
                        input={<OutlinedInput id="select-multiple-chip" label="Select Members" />}
                        renderValue={(selected) => (
                            <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
                                {selected.map((value) => {
                                    // Find the friend object to display their name
                                    const friend = friends.find(f => f.PeerID === value);
                                    return (
                                        <Chip
                                            key={value}
                                            label={friend ? (friend.display_name || value) : value}
                                        />
                                    );
                                })}
                            </Box>
                        )}
                    >
                        {availableFriends.map((friend) => (
                            <MenuItem
                                key={friend.PeerID}
                                value={friend.PeerID}
                                // style={getStyles(name, personName, theme)}
                            >
                                {friend.display_name || friend.PeerID}
                            </MenuItem>
                        ))}
                    </Select>
                </FormControl>
            </DialogContent>
            <DialogActions>
                <Button onClick={onClose} color="secondary">Cancel</Button>
                <Button onClick={handleCreate} color="primary" disabled={loading}>
                    {loading ? 'Creating...' : 'Create'}
                </Button>
            </DialogActions>
        </Dialog>
    );
};

export default CreateGroupChat;