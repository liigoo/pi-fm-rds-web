// 全局状态管理
const AppState = {
    data: {
        frequency: 88.0,
        isRunning: false,
        isPaused: false,
        playlist: [],
        files: [],
        selectedFileID: '',
        currentTrack: -1,
        currentFileID: '',
        connectionStatus: 'disconnected',
        spectrumData: []
    },

    listeners: [],

    subscribe(listener) {
        this.listeners.push(listener);
    },

    setState(updates) {
        Object.assign(this.data, updates);
        this.notify();
    },

    notify() {
        this.listeners.forEach((listener) => {
            try {
                listener(this.data);
            } catch (err) {
                console.error('State listener error:', err);
            }
        });
    },

    getState() {
        return { ...this.data };
    }
};

window.AppState = AppState;
