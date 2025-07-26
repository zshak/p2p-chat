export const getPeerId = () => {
    return localStorage.getItem('peerID') || '';
};

export const setPeerId = (peerId) => {
    localStorage.setItem('peerID', peerId);
};