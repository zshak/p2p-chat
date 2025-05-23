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

function LoginPage() {
    const [daemonState, setDaemonState] = useState(null);
    const [password, setPassword] = useState('');
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(null);
    const [loginStep, setLoginStep] = useState(LOGIN_STEPS.INITIAL);
    const navigate = useNavigate();
    const theme = useTheme();
    const isMobile = useMediaQuery(theme.breakpoints.down('sm'));

    useEffect(() => {
        // Check daemon status when component mounts
        checkDaemonStatus();
    }, []);

    const checkDaemonStatus = async () => {
        try {
            setLoading(true);
            const response = await checkStatus();
            setDaemonState(response.data.state);

            if (response.data.state === DAEMON_STATES.RUNNING) {
                navigate('/chat');
            }

            setLoading(false);
        } catch (err) {
            setError('Failed to connect to the server. Please make sure the backend is running.');
            setLoading(false);
        }
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
            setLoginStep(LOGIN_STEPS.SENDING_REQUEST);

            try {
                await unlockWithPassword(password);

                setTimeout(async () => {
                    setLoginStep(LOGIN_STEPS.SETUP_CONNECTION);

                    setTimeout(async () => {
                        setLoginStep(LOGIN_STEPS.VERIFYING_NETWORK);

                        try {
                            const statusResponse = await checkStatus();
                            setDaemonState(statusResponse.data.state);

                            setTimeout(() => {
                                setLoginStep(LOGIN_STEPS.SUCCESS);

                                setTimeout(() => {
                                    navigate('/chat');
                                }, 200);
                            }, 500);

                        } catch (statusErr) {
                            setTimeout(() => {
                                setLoginStep(LOGIN_STEPS.SUCCESS);
                                navigate('/chat');
                            }, 300);
                        }
                    }, 900);
                }, 600);

            } catch (unlockErr) {
                setError('Invalid password. Please try again.');
                setLoading(false);
                setPassword('');
                setLoginStep(LOGIN_STEPS.INITIAL);
            }

        } catch (err) {
            setError('Invalid password. Please try again.');
            setLoading(false);
            setPassword('');
            setLoginStep(LOGIN_STEPS.INITIAL);
        }
    };

    const handleRegister = async () => {
        try {
            setLoading(true);
            await registerUser();
            await checkDaemonStatus();
            setLoading(false);
        } catch (err) {
            setError('Registration failed. Please try again.');
            setLoading(false);
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

                {daemonState === DAEMON_STATES.INITIALIZING && (
                    <Box sx={{ width: '100%' }}>
                        <LoginForm
                            password={password}
                            setPassword={setPassword}
                            handlePasswordSubmit={handlePasswordSubmit}
                            error={error}
                            isMobile={isMobile}
                        >
                            <RegisterForm handleRegister={handleRegister} />
                        </LoginForm>
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