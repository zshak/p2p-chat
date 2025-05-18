import React from 'react';
import {
    Box,
    Typography,
    List,
    ListItem,
    ListItemText,
    Avatar,
    Divider,
    Badge,
    Button,
} from '@mui/material';
import PersonIcon from '@mui/icons-material/Person';
import SettingsIcon from '@mui/icons-material/Settings';
import LogoutIcon from '@mui/icons-material/Logout';
import { useNavigate } from 'react-router-dom';

const Sidebar = ({ activeUsers }) => {
    const navigate = useNavigate();

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
                    <PersonIcon fontSize="large" />
                </Avatar>
                <Typography variant="h6" color="primary.dark">
                    Your Name
                </Typography>
                <Typography variant="body2" color="text.secondary">
                    Online
                </Typography>
            </Box>

            <Divider />

            <Box sx={{ p: 2 }}>
                <Typography variant="subtitle2" color="text.secondary" sx={{ pl: 1, mb: 1 }}>
                    ACTIVE CONTACTS
                </Typography>
                <List>
                    {activeUsers.map((user) => (
                        <ListItem
                            button
                            key={user.id}
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
                                color={user.status === 'online' ? 'success' : 'error'}
                                variant="dot"
                                anchorOrigin={{
                                    vertical: 'bottom',
                                    horizontal: 'right',
                                }}
                                overlap="circular"
                                sx={{ mr: 2 }}
                            >
                                <Avatar sx={{ bgcolor: 'secondary.light' }}>
                                    {user.name.charAt(0)}
                                </Avatar>
                            </Badge>
                            <ListItemText
                                primary={user.name}
                                primaryTypographyProps={{
                                    noWrap: true,
                                    fontSize: 14,
                                    fontWeight: 500
                                }}
                            />
                        </ListItem>
                    ))}
                </List>
            </Box>

            <Box sx={{ flexGrow: 1 }} />

            <Box sx={{ p: 2 }}>
                <Button
                    fullWidth
                    variant="outlined"
                    color="primary"
                    startIcon={<SettingsIcon />}
                    sx={{ mb: 1 }}
                >
                    Settings
                </Button>
                <Button
                    fullWidth
                    variant="contained"
                    color="error"
                    startIcon={<LogoutIcon />}
                    onClick={() => navigate('/login')}
                >
                    Logout
                </Button>
            </Box>
        </>
    );
};

export default Sidebar;