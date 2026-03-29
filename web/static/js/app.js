// 主应用入口
const App = {
    wsClient: null,
    pollingTimer: null,

    async init() {
        try {
            this.wsClient = new WSClient();
            this.wsClient.connect();

            Controls.init();
            Files.init();
            Playlist.init();
            Spectrum.init();

            await this.loadSnapshot();
            this.startPolling();
        } catch (err) {
            console.error('Initialization error:', err);
            alert('应用初始化失败: ' + err.message);
        }
    },

    async loadSnapshot() {
        const [status, playlist, files] = await Promise.all([
            API.getStatus(),
            API.getPlaylist(),
            API.getFiles()
        ]);

        const playlistItems = Array.isArray(playlist?.items) ? playlist.items : [];
        const fileItems = Array.isArray(files?.files) ? files.files : [];

        AppState.setState({
            frequency: status?.frequency || 88.0,
            isRunning: !!status?.running,
            isPaused: !!status?.paused,
            currentFileID: status?.current_file || '',
            currentTrack: Number.isInteger(status?.current_index) ? status.current_index : -1,
            playlist: playlistItems,
            files: fileItems
        });
    },

    startPolling() {
        if (this.pollingTimer) {
            clearInterval(this.pollingTimer);
        }

        this.pollingTimer = setInterval(async () => {
            try {
                await this.loadSnapshot();
            } catch (err) {
                console.error('Polling failed:', err);
            }
        }, 1500);
    },

    destroy() {
        if (this.pollingTimer) {
            clearInterval(this.pollingTimer);
            this.pollingTimer = null;
        }

        if (this.wsClient) {
            this.wsClient.disconnect();
            this.wsClient = null;
        }
    }
};

window.App = App;

if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', () => App.init());
} else {
    App.init();
}

window.addEventListener('beforeunload', () => {
    App.destroy();
});
