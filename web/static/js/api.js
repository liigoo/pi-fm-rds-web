// API 调用封装
const API = {
    baseURL: '/api',

    // 通用请求方法
    async request(endpoint, options = {}) {
        try {
            const response = await fetch(`${this.baseURL}${endpoint}`, {
                headers: {
                    'Content-Type': 'application/json',
                    ...options.headers
                },
                ...options
            });

            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }

            return await response.json();
        } catch (err) {
            console.error('API Error:', err);
            throw err;
        }
    },

    // 获取状态
    async getStatus() {
        return this.request('/status');
    },

    // 设置频率
    async setFrequency(frequency) {
        return this.request('/frequency', {
            method: 'POST',
            body: JSON.stringify({ frequency })
        });
    },

    // 启动传输
    async start() {
        return this.request('/broadcast/start', { method: 'POST' });
    },

    // 停止传输
    async stop() {
        return this.request('/broadcast/stop', { method: 'POST' });
    },

    // 上传文件
    async uploadFile(file) {
        const formData = new FormData();
        formData.append('file', file);

        const response = await fetch(`${this.baseURL}/files/upload`, {
            method: 'POST',
            body: formData
        });

        if (!response.ok) {
            throw new Error(`Upload failed: ${response.statusText}`);
        }

        return await response.json();
    },

    // 获取播放列表
    async getPlaylist() {
        return this.request('/playlist');
    },

    // 添加到播放列表
    async addToPlaylist(fileID, filename) {
        return this.request('/playlist/add', {
            method: 'POST',
            body: JSON.stringify({ file_id: fileID, filename })
        });
    },

    // 播放指定曲目
    async playTrack(index) {
        return this.request('/playlist/play', {
            method: 'POST',
            body: JSON.stringify({ index })
        });
    }
};

window.API = API;
