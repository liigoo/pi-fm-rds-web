// 全局状态管理
const AppState = {
    data: {
        frequency: 88.0,
        isRunning: false,
        playlist: [],
        currentTrack: -1,
        connectionStatus: 'disconnected',
        spectrumData: []
    },

    listeners: [],

    // 订阅状态变化
    subscribe(listener) {
        this.listeners.push(listener);
    },

    // 更新状态
    setState(updates) {
        Object.assign(this.data, updates);
        this.notify();
    },

    // 通知所有监听器
    notify() {
        this.listeners.forEach(listener => {
            try {
                listener(this.data);
            } catch (err) {
                console.error('State listener error:', err);
            }
        });
    },

    // 获取状态
    getState() {
        return { ...this.data };
    }
};

window.AppState = AppState;
