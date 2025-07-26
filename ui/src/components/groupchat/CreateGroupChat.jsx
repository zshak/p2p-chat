import React, {useState} from 'react';
import {
    Box,
    Button,
    Chip,
    Dialog,
    DialogActions,
    DialogContent,
    DialogTitle,
    FormControl,
    InputLabel,
    MenuItem,
    OutlinedInput,
    Select,
    TextField,
    Typography
} from '@mui/material';
import {createGroupChat} from '../../services/api';
import {getPeerId} from "../utils/userStore";

const CreateGroupChat = ({open, onClose, onCreateGroupChat, friends}) => {
    const [groupName, setGroupName] = useState('');
    const [selectedMembers, setSelectedMembers] = useState([]);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState('');
    const ownPeerId = getPeerId();

    const handleCreate = async () => {
        if (!groupName.trim() || selectedMembers.length === 0) {
            setError('Group name and members are required.');
            return;
        }
        setLoading(true);
        setError('');
        const allMembers = [...selectedMembers];
        try {
            await createGroupChat(allMembers, groupName);
            if (onCreateGroupChat) {
                onCreateGroupChat();
            }
            onClose();
            setGroupName('');
            setSelectedMembers([]);
        } catch (err) {
            console.error('Failed to create group chat:', err);
            setError('Failed to create group chat. Please try again.');
        } finally {
            setLoading(false);
        }
    };

    const handleMemberSelect = (event) => {
        const {
            target: {value},
        } = event;
        setSelectedMembers(
            typeof value === 'string' ? value.split(',') : value,
        );
    };

    const availableFriends = friends.filter(friend => friend.PeerID !== ownPeerId);

    return (
        <Dialog open={open} onClose={onClose} fullWidth maxWidth="sm">
            <DialogTitle>Create New Group Chat</DialogTitle>
            <DialogContent>
                {error && (
                    <Typography color="error" variant="body2" sx={{mb: 2}}>
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
                    sx={{mb: 2}}
                />
                <FormControl fullWidth>
                    <InputLabel id="select-members-label">Select Members</InputLabel>
                    <Select
                        labelId="select-members-label"
                        multiple
                        value={selectedMembers}
                        onChange={handleMemberSelect}
                        input={<OutlinedInput id="select-multiple-chip" label="Select Members"/>}
                        renderValue={(selected) => (
                            <Box sx={{display: 'flex', flexWrap: 'wrap', gap: 0.5}}>
                                {selected.map((value) => {
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