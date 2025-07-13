import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import {
    Container,
    Box,
    Typography,
    useTheme,
    useMediaQuery
} from '@mui/material';
import { checkStatus, registerUser, unlockWithPassword } from "../../services/api.js";
import LoadingScreen from './LoadingScreen';
import LoginForm from './LoginForm';
import RegisterForm from '../register/RegisterForm';
import { LOGIN_STEPS, LOGIN_STEP_MESSAGES, DAEMON_STATES } from '../utils/constants';
import {setPeerId} from '../utils/userStore.js'

function LoginPage() {
    const [daemonState, setDaemonState] = useState(null);
    const [peerIdReceived, setPeerIdReceived] = useState(false);
    const [password, setPassword] = useState('');
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(null);
    const [loginStep, setLoginStep] = useState(LOGIN_STEPS.INITIAL);
    const navigate = useNavigate();
    const theme = useTheme();
    const isMobile = useMediaQuery(theme.breakpoints.down('sm'));

    useEffect(() => {
        checkDaemonStatus();
    }, []);

    const checkDaemonStatus = async () => {
        try {
            console.log('check daemon status');
            setLoading(true);
            const response = await checkStatus();
            setDaemonState(response.data.state);

            if (response.data.state === DAEMON_STATES.RUNNING) {
                setPeerId(response.data.peer_id)
                console.log('set peer id: ' + response.data.peer_id);
                navigate('/chat');
            }

            setLoading(false);
        } catch (err) {
            setError('Failed to connect to the server. Please make sure the backend is running.');
            setLoading(false);
        }
    };

    const pollStatusWithRandomSteps = async () => {
        const steps = [
            LOGIN_STEPS.SENDING_REQUEST,
            LOGIN_STEPS.SETUP_CONNECTION,
            LOGIN_STEPS.VERIFYING_NETWORK
        ];

        let currentStepIndex = 0;
        setLoginStep(steps[currentStepIndex]);

        const pollInterval = setInterval(async () => {
            try {
                const response = await checkStatus();
                const { state, peer_id } = response.data;

                // Check if we received peer_id for the first time
                if (peer_id && !peerIdReceived) {
                    setPeerIdReceived(true);
                    // Move to next random step when peer_id is received
                    currentStepIndex = Math.min(currentStepIndex + 1, steps.length - 1);
                    setLoginStep(steps[currentStepIndex]);
                }

                // Check if state is "Running"
                if (state === DAEMON_STATES.RUNNING) {
                    setPeerId(response.data.peer_id)
                    clearInterval(pollInterval);
                    setLoginStep(LOGIN_STEPS.SUCCESS);

                    setTimeout(() => {
                        navigate('/chat');
                    }, 1000);
                    return;
                }

                // Randomly advance to next step occasionally (to show progress)
                if (Math.random() < 0.3 && currentStepIndex < steps.length - 1) {
                    currentStepIndex++;
                    setLoginStep(steps[currentStepIndex]);
                }

            } catch (error) {
                console.error('Error polling status:', error);
                // Continue polling even on error
            }
        }, 1500); // Poll every 1.5 seconds

        // Cleanup interval after 30 seconds max to prevent infinite polling
        setTimeout(() => {
            clearInterval(pollInterval);
            if (loading) {
                setError('Connection timeout. Please try again.');
                setLoading(false);
                setLoginStep(LOGIN_STEPS.INITIAL);
            }
        }, 30000);
    };

    const handlePasswordSubmit = async (e) => {
        e.preventDefault();
        if (!password.trim()) {
            setError('Password cannot be empty');
            return;
        }

        try {
            setError(null);
            setLoading(true);
            setPeerIdReceived(false);

            await unlockWithPassword(password);

            // Start polling with random UI steps
            pollStatusWithRandomSteps();

        } catch (unlockErr) {
            setError('Invalid password. Please try again.');
            setLoading(false);
            setPassword('');
            setLoginStep(LOGIN_STEPS.INITIAL);
        }
    };

    const handleRegister = async (password) => {
        try {
            setError(null);
            setLoading(true);
            setPeerIdReceived(false);

            await registerUser(password);

            // Start polling with random UI steps
            pollStatusWithRandomSteps();

        } catch (err) {
            setError(err.response?.data || 'Registration failed. Please try again.');
            setLoading(false);
            setLoginStep(LOGIN_STEPS.INITIAL);
        }
    };

    if (loading) {
        const stepMessage = LOGIN_STEP_MESSAGES[loginStep] || {
            title: 'Loading P2P Chat...',
            subtitle: 'Please wait'
        };

        return (
            <LoadingScreen
                loginStep={loginStep}
                message={stepMessage.title}
                subMessage={stepMessage.subtitle}
            />
        );
    }

    return (
        <Container
            component="main"
            maxWidth="sm"
            sx={{
                minHeight: '100vh',
                display: 'flex',
                flexDirection: 'column',
                justifyContent: 'center'
            }}
        >
            <Box sx={{
                display: 'flex',
                flexDirection: 'column',
                alignItems: 'center',
                width: '100%',
            }}>
                {daemonState === DAEMON_STATES.WAITING_FOR_PASSWORD && (
                    <LoginForm
                        password={password}
                        setPassword={setPassword}
                        handlePasswordSubmit={handlePasswordSubmit}
                        error={error}
                        isMobile={isMobile}
                    />
                )}

                {daemonState === DAEMON_STATES.WAITING_FOR_KEY && (
                    <Box sx={{ width: '100%' }}>
                        <RegisterForm
                            handleRegister={handleRegister}
                            error={error}
                            loading={loading}
                        />
                    </Box>
                )}

                <Typography variant="body2" color="text.secondary" align="center" sx={{ mt: 3 }}>
                    © {new Date().getFullYear()} P2P Chat - Bachelor's Project
                </Typography>
            </Box>
        </Container>
    );
}

export default LoginPage;