// 主应用入口
(function() {
    'use strict';

    let wsClient = null;

    // 初始化应用
    async function init() {
        try {
            // 初始化状态管理
            console.log('Initializing state...');

            // 初始化 WebSocket
            console.log('Connecting WebSocket...');
            wsClient = new WSClient();
            wsClient.connect();

            // 初始化控制面板
            console.log('Initializing controls...');
            Controls.init();

            // 初始化播放列表
            console.log('Initializing playlist...');
            Playlist.init();

            // 初始化频谱可视化
            console.log('Initializing spectrum...');
            Spectrum.init();

            // 加载初始数据
            console.log('Loading initial data...');
            await loadInitialData();

            console.log('Application initialized successfully');
        } catch (err) {
            console.error('Initialization error:', err);
            alert('应用初始化失败: ' + err.message);
        }
    }

    // 加载初始数据
    async function loadInitialData() {
        try {
            const [status, playlist] = await Promise.all([
                API.getStatus(),
                API.getPlaylist()
            ]);

            AppState.setState({
                frequency: status.frequency || 88.0,
                isRunning: status.running || false,
                playlist: playlist || []
            });
        } catch (err) {
            console.error('Failed to load initial data:', err);
        }
    }

    // 更新连接状态显示
    AppState.subscribe((state) => {
        const statusEl = document.getElementById('connectionStatus');
        if (!statusEl) return;

        const dot = statusEl.querySelector('.status-dot');
        const text = statusEl.querySelector('.status-text');

        if (state.connectionStatus === 'connected') {
            dot.className = 'status-dot connected';
            text.textContent = '已连接';
        } else {
            dot.className = 'status-dot disconnected';
            text.textContent = '未连接';
        }
    });

    // 全局错误处理
    window.addEventListener('error', (event) => {
        console.error('Global error:', event.error);
    });

    window.addEventListener('unhandledrejection', (event) => {
        console.error('Unhandled promise rejection:', event.reason);
    });

    // DOM 加载完成后初始化
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }

    // 页面卸载时清理
    window.addEventListener('beforeunload', () => {
        if (wsClient) {
            wsClient.disconnect();
        }
    });
})();
