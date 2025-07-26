import axios from 'axios';

// const API_BASE_URL = ("http://" + import.meta.env.VITE_BACKEND_API_BASE_URL) || 'http://127.0.0.1:59578';

const API_BASE_URL = import.meta.env.VITE_BACKEND_API_BASE_URL || `${window.location.origin}/api`;


const api = axios.create({
    baseURL: API_BASE_URL,
    timeout: 10000,
    headers: {
        'Content-Type': 'application/json',
    }
});

// register + login endpoints
export const checkStatus = () => api.get('/status');
export const unlockWithPassword = (password) => api.post('/setup/unlock-key', {password});
export const registerUser = (password) => api.post('/setup/create-key', {password});

// friends request endpoints
export const sendFriendRequest = (receiver_peer_id) => api.post('/profile/friend/request', {receiver_peer_id});
export const respondToFriendRequest = (peer_id, is_accepted) => api.patch('/profile/friend/response', {
    peer_id,
    is_accepted
});
export const getFriends = () => api.get('/profile/friends');
export const getFriendRequests = () => api.get('/profile/friendRequests');

// Group chat endpoints
export const getGroupChats = () => api.get('/group-chats');
export const getGroupChatMessages = (group_id) => api.post('/group-chat/messages', {group_id});
export const createGroupChat = (member_peers, name) => api.post('/group-chat', {member_peers, name});

export const getChatMessages = (peer_id) => api.post('/chat/messages', {peer_id});

export const setDisplayNameAPI = async (entityId, entityType, displayName) => {
    try {
        const response = await fetch(`${API_BASE_URL}/profile/display-name`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                entity_id: entityId,
                entity_type: entityType,
                display_name: displayName
            }),
        });

        if (!response.ok) {
            const errorText = await response.text();
            throw new Error(errorText || 'Failed to set display name');
        }

        return response;
    } catch (error) {
        console.error(`Failed to set display name for ${entityType} ${entityId}:`, error);
        throw error;
    }
};

export const getDisplayNameAPI = async (entityId, entityType) => {
    try {
        const response = await fetch(`${API_BASE_URL}/profile/display-name/get`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                entity_id: entityId,
                entity_type: entityType
            }),
        });

        if (!response.ok) {
            console.warn(`Failed to get display name for ${entityType} ${entityId}, using fallback`);
            return {
                entity_id: entityId,
                entity_type: entityType,
                display_name: formatEntityIdFallback(entityId, entityType),
                is_custom_name: false
            };
        }

        const data = await response.json();
        return data;
    } catch (error) {
        console.warn(`Error getting display name for ${entityType} ${entityId}:`, error);
        return {
            entity_id: entityId,
            entity_type: entityType,
            display_name: formatEntityIdFallback(entityId, entityType),
            is_custom_name: false
        };
    }
};

export const deleteDisplayNameAPI = async (entityId, entityType) => {
    try {
        const response = await fetch(`${API_BASE_URL}/profile/display-name/delete`, {
            method: 'DELETE',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                entity_id: entityId,
                entity_type: entityType
            }),
        });
        if (!response.ok) {
            const errorText = await response.text();
            if (response.status === 404) {
                console.log(`No display name to delete for ${entityType} ${entityId}`);
                return response;
            }
            throw new Error(errorText || 'Failed to delete display name');
        }

        return response;
    } catch (error) {
        console.error(`Failed to delete display name for ${entityType} ${entityId}:`, error);
        throw error;
    }
};

const formatEntityIdFallback = (entityId, entityType) => {
    if (!entityId) return 'Unknown';
    if (entityType === 'group') {
        return 'Group Chat';
    }
    if (entityId.length >= 8) {
        const first2 = entityId.substring(0, 2);
        const last6 = entityId.substring(entityId.length - 6);
        return `${first2}*${last6}`;
    }

    return entityId;
};

export default api;