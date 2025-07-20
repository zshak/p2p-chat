import React, {useEffect, useState} from 'react';
import {
    Alert,
    Box,
    Button,
    Dialog,
    DialogActions,
    DialogContent,
    DialogTitle,
    IconButton,
    TextField,
    Typography
} from '@mui/material';
import EditIcon from '@mui/icons-material/Edit';
import DeleteIcon from '@mui/icons-material/Delete';
import {setDisplayNameAPI, getDisplayNameAPI, deleteDisplayNameAPI} from "../../services/api.js"

const EditDisplayName = ({entity, entityType, currentDisplayName, onUpdate, onClose}) => {
    const [open, setOpen] = useState(false);
    const [displayName, setDisplayName] = useState('');
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState('');
    const [success, setSuccess] = useState('');

    useEffect(() => {
        if (open) {
            setDisplayName(currentDisplayName || '');
            setError('');
            setSuccess('');
        }
    }, [open, currentDisplayName]);

    const handleOpen = () => {
        setOpen(true);
    };

    const handleClose = () => {
        setOpen(false);
        if (onClose) onClose();
    };

    const handleSave = async () => {
        if (!displayName.trim()) {
            setError('Display name cannot be empty');
            return;
        }

        setLoading(true);
        setError('');
        setSuccess('');

        try {
            console.log('Saving display name:', {
                entityId: entity.PeerID || entity.group_id,
                entityType: entityType,
                displayName: displayName.trim()
            });

            await setDisplayNameAPI(entity.PeerID || entity.group_id, entityType, displayName.trim());
            setSuccess('Display name updated successfully!');

            if (onUpdate) {
                onUpdate(displayName.trim());
            }

            setTimeout(() => {
                setSuccess('');
                handleClose();
            }, 1500);
        } catch (err) {
            console.error('Failed to update display name:', err);
            const errorMessage = err.response?.data || err.message || 'Failed to update display name';
            setError(errorMessage);
        } finally {
            setLoading(false);
        }
    };

    const handleDelete = async () => {
        setLoading(true);
        setError('');
        setSuccess('');

        try {
            console.log('Deleting display name:', {
                entityId: entity.PeerID || entity.group_id,
                entityType: entityType
            });

            await deleteDisplayNameAPI(entity.PeerID || entity.group_id, entityType);
            setSuccess('Display name removed successfully!');

            if (onUpdate) {
                onUpdate('');
            }

            setTimeout(() => {
                setSuccess('');
                handleClose();
            }, 1500);
        } catch (err) {
            console.error('Failed to delete display name:', err);
            const errorMessage = err.response?.data || err.message || 'Failed to delete display name';
            setError(errorMessage);
        } finally {
            setLoading(false);
        }
    };

    const formatEntityId = (id) => {
        if (!id || id.length < 8) return id;
        return `${id.substring(0, 4)}...${id.substring(id.length - 4)}`;
    };

    return (
        <>
            <IconButton
                size="small"
                onClick={handleOpen}
                sx={{
                    opacity: 0.7,
                    '&:hover': {
                        opacity: 1,
                        backgroundColor: 'primary.light',
                        color: 'primary.contrastText'
                    }
                }}
            >
                <EditIcon fontSize="small"/>
            </IconButton>

            <Dialog open={open} onClose={handleClose} maxWidth="sm" fullWidth>
                <DialogTitle>
                    <Box sx={{display: 'flex', alignItems: 'center', gap: 1}}>
                        <EditIcon color="primary"/>
                        Edit Display Name
                    </Box>
                </DialogTitle>

                <DialogContent>
                    <Typography variant="body2" color="text.secondary" sx={{mb: 2}}>
                        Set a custom display name for {entityType} {formatEntityId(entity.PeerID || entity.group_id)}
                    </Typography>

                    {error && <Alert severity="error" sx={{mb: 2}}>{error}</Alert>}
                    {success && <Alert severity="success" sx={{mb: 2}}>{success}</Alert>}

                    <TextField
                        autoFocus
                        margin="dense"
                        label="Display Name"
                        fullWidth
                        variant="outlined"
                        value={displayName}
                        onChange={(e) => setDisplayName(e.target.value)}
                        disabled={loading}
                        placeholder={`Enter display name for ${entityType}...`}
                        sx={{mb: 2}}
                    />

                    {currentDisplayName && (
                        <Box sx={{
                            display: 'flex',
                            alignItems: 'center',
                            gap: 1,
                            p: 1,
                            backgroundColor: 'grey.100',
                            borderRadius: 1
                        }}>
                            <Typography variant="body2" color="text.secondary">
                                Current: {currentDisplayName}
                            </Typography>
                            <IconButton
                                size="small"
                                onClick={handleDelete}
                                disabled={loading}
                                color="error"
                                title="Remove display name"
                            >
                                <DeleteIcon fontSize="small"/>
                            </IconButton>
                        </Box>
                    )}
                </DialogContent>

                <DialogActions>
                    <Button onClick={handleClose} disabled={loading}>
                        Cancel
                    </Button>
                    <Button
                        onClick={handleSave}
                        variant="contained"
                        disabled={loading || !displayName.trim()}
                    >
                        {loading ? 'Saving...' : 'Save'}
                    </Button>
                </DialogActions>
            </Dialog>
        </>
    );
};

export default EditDisplayName;