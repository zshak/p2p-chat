import React from 'react';
import {
    Paper,
    TextField,
    IconButton,
} from '@mui/material';
import SendIcon from '@mui/icons-material/Send';

const MessageInput = ({ newMessage, setNewMessage, handleSendMessage }) => {
    return (
        <Paper
            component="form"
            onSubmit={handleSendMessage}
            sx={{
                p: 2,
                display: 'flex',
                alignItems: 'center',
                borderTop: '1px solid',
                borderColor: 'divider',
                borderRadius: 0,
            }}
            elevation={0}
        >
            <TextField
                fullWidth
                variant="outlined"
                placeholder="Type your message..."
                value={newMessage}
                onChange={(e) => setNewMessage(e.target.value)}
                sx={{
                    '& .MuiOutlinedInput-root': {
                        borderRadius: 4,
                    },
                }}
            />
            <IconButton
                type="submit"
                color="primary"
                sx={{
                    ml: 1,
                    bgcolor: 'primary.main',
                    color: 'white',
                    '&:hover': {
                        bgcolor: 'primary.dark'
                    }
                }}
            >
                <SendIcon />
            </IconButton>
        </Paper>
    );
};

export default MessageInput;