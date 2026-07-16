

Readme · MD
# P2P Chat
 
A fully decentralised, zero-registration messenger. There is no server, no account, and no email or phone number — a user's identity **is** a cryptographic key pair. Peers discover each other over a distributed hash table, connect directly (punching through NATs where needed), and exchange end-to-end encrypted messages.
 
Built as a Bachelor's thesis project for the Free University of Tbilisi (MACS).
 
> **Status:** academic / proof-of-concept. Not security-audited for production use.
 
---
 
## Why it exists
 
Mainstream messengers tie your identity to a phone number and route every message through central infrastructure that can be logged, blocked, or compelled to hand over data. This project explores the opposite design point: **no central authority anywhere in the system.** Identity, discovery, connectivity, and encryption are all handled peer-to-peer.
 
## Key features
 
- **Keypair-as-identity** — each user is an Ed25519 key pair. No registration, no personal data, nothing to leak. The public key is the address.
- **Serverless peer discovery** — peers are located through a Kademlia distributed hash table (and mDNS on the local network), so there is no directory server to take down or trust.
- **Direct connectivity through NATs** — transports over TCP, QUIC and WebRTC, with hole punching and relay fallback, so two peers behind home routers can still reach each other.
- **End-to-end encryption** — transport security via the Noise protocol / TLS; group messages encrypted with AES-256-GCM.
- **Encrypted keys at rest** — private keys are sealed with a key derived from the user's passphrase via Argon2id, so a stolen disk does not surrender the identity.
## Tech stack
 
| Layer            | Technology                                                    |
| ---------------- | ------------------------------------------------------------- |
| P2P networking   | Go, [libp2p](https://libp2p.io/) — DHT Kademlia, Noise, multi-transport |
| Backend / daemon | Go                                                            |
| Frontend         | React (JavaScript)                                            |
| Cryptography     | Ed25519, AES-256-GCM, Argon2id                                |
 
## Architecture
 
The project separates the peer-to-peer engine from the interface: a headless **daemon** owns all networking, discovery, and encryption, and a **UI** connects to it. Keeping the two apart means the networking layer can run and be tested on its own, without the frontend.
 
```
p2p-chat/
├── daemon/    # Go networking node — libp2p host, DHT peer discovery, transports, encryption
├── ui/        # React frontend
├── scripts/   # helper scripts for building and running the project
└── files/     # supporting project files
```
