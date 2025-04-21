const WebSocket = require('ws');
const readline = require('readline');

// Create WebSocket connection
// const socket = new WebSocket('ws://localhost:59578/ws');
const socket = new WebSocket('ws://localhost:59579/ws');

// Create interface for console input
const rl = readline.createInterface({
    input: process.stdin,
    output: process.stdout
});

// Default peer ID
// const DEFAULT_PEER_ID = '12D3KooWCvSxbpi2y6nx3QbnjBovxi1LP3MvvZ5eGqezuGty8iUg';
const DEFAULT_PEER_ID = '12D3KooWSdPV93Lu69xwaWSocfBLYp5UbQNQ8e1usjXQZpVrcatU';

// Connection opened
socket.on('open', () => {
    console.log('Connection established');
    askForInput();
});

// Listen for messages
socket.on('message', (data) => {
    console.log(`Received: ${data}`);
});

// Connection closed
socket.on('close', () => {
    console.log('Connection closed');
    rl.close();
});

// Connection error
socket.on('error', (error) => {
    console.error(`WebSocket error: ${error.message}`);
});

function askForInput() {
    rl.question('Enter message: ', (message) => {
        if (message.toLowerCase() === 'exit') {
            socket.close();
            rl.close();
            return;
        }

        const payload = {
            target_peer_id: DEFAULT_PEER_ID,
            message: message
        };

        socket.send(JSON.stringify(payload));
        askForInput(); // Continue asking for input
    });
}

// npm init -y
// npm install ws
// node test.js