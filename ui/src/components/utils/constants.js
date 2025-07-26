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
        subtitle: 'Processing your request'
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
        title: 'Setup successful!',
        subtitle: 'Redirecting to chat...'
    }
};

export const DAEMON_STATES = {
    WAITING_FOR_PASSWORD: 'Waiting for Password via API',
    WAITING_FOR_KEY: 'Waiting for Key Setup via API',
    RUNNING: 'Running',
    INITIALIZING: 'Initializing P2P Network'
};