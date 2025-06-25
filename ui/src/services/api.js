import axios from 'axios';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://127.0.0.1:59579';

const api = axios.create({
    baseURL: API_BASE_URL,
    timeout: 10000,
    headers: {
        'Content-Type': 'application/json',
    }
});

// register + login endpoints
export const checkStatus = () => api.get('/status');
export const unlockWithPassword = (password) => api.post('/setup/unlock-key', { password });
export const registerUser = (password) => api.post('/setup/create-key', { password });

// friends request endpoints
export const sendFriendRequest = (receiverPeerId) => api.post('/profile/friend/request', { receiverPeerId });
export const respondToFriendRequest = (peerId, isAccepted) => api.patch('/profile/friend/response', { peerId, isAccepted });
export const getFriends = () => api.get('/profile/friend');

export default api;