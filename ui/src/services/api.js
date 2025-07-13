import axios from 'axios';

const API_BASE_URL = ("http://" + import.meta.env.VITE_BACKEND_API_BASE_URL) || 'http://127.0.0.1:59578';

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
export const sendFriendRequest = (receiver_peer_id) => api.post('/profile/friend/request', { receiver_peer_id });
export const respondToFriendRequest = (peer_id, is_accepted) => api.patch('/profile/friend/response', { peer_id, is_accepted });
export const getFriends = () => api.get('/profile/friends');
export const getFriendRequests = () => api.get('/profile/friendRequests');

// Group chat endpoints
export const getGroupChats = () => api.get('/group-chats');
export const getGroupChatMessages = (group_id) => api.post('/group-chat/messages', { group_id });
export const createGroupChat = (member_peers, name) => api.post('/group-chat', { member_peers, name });

export default api;