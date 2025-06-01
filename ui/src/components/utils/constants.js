export const LOGIN_STEPS = {
    INITIAL: 0,
    SENDING_REQUEST: 1,
    SETUP_CONNECTION: 2,
    VERIFYING_NETWORK: 3,
    SUCCESS: 4
};

export const LOGIN_STEP_MESSAGES = {
    [LOGIN_STEPS.SENDING_REQUEST]: {
        title: 'Sending request...',
        subtitle: 'Unlocking with your password'
    },
    [LOGIN_STEPS.SETUP_CONNECTION]: {
        title: 'Setting up secure connection...',
        subtitle: 'Initializing P2P network'
    },
    [LOGIN_STEPS.VERIFYING_NETWORK]: {
        title: 'Verifying P2P network...',
        subtitle: 'Checking connection status'
    },
    [LOGIN_STEPS.SUCCESS]: {
        title: 'Login successful!',
        subtitle: 'Redirecting to chat...'
    }
};

export const SAMPLE_MESSAGES = [
    { id: 1, sender: 'other', text: 'Hello! How are you today?', timestamp: new Date().setMinutes(new Date().getMinutes() - 60) },
    { id: 2, sender: 'me', text: 'Hi there! I\'m doing well, thanks for asking. How about you?', timestamp: new Date().setMinutes(new Date().getMinutes() - 55) },
    { id: 3, sender: 'other', text: 'I\'m great! Just working on this P2P Chat implementation.', timestamp: new Date().setMinutes(new Date().getMinutes() - 30) },
    { id: 4, sender: 'me', text: 'That sounds interesting! How is it going so far?', timestamp: new Date().setMinutes(new Date().getMinutes() - 28) },
    { id: 5, sender: 'other', text: 'It\'s challenging but fun. I\'m learning a lot about WebSockets and P2P communication.', timestamp: new Date().setMinutes(new Date().getMinutes() - 25) },
];

export const ACTIVE_USERS = [
    { id: 1, name: 'Alice', status: 'online' },
    { id: 2, name: 'Bob', status: 'online' },
    { id: 3, name: 'Charlie', status: 'offline' },
];

export const DAEMON_STATES = {
    WAITING_FOR_PASSWORD: 'Waiting for Password via API',
    RUNNING: 'Running',
    INITIALIZING: 'Initializing P2P Network'
};