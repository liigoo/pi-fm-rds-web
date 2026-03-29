// API 调用封装
const API = {
    baseURL: '/api',

    async request(endpoint, options = {}) {
        const headers = { ...(options.headers || {}) };
        const hasFormData = options.body instanceof FormData;

        if (!hasFormData && !headers['Content-Type']) {
            headers['Content-Type'] = 'application/json';
        }

        const response = await fetch(`${this.baseURL}${endpoint}`, {
            ...options,
            headers
        });

        const contentType = response.headers.get('Content-Type') || '';
        const isJSON = contentType.includes('application/json');
        const payload = isJSON ? await response.json() : await response.text();

        if (!response.ok) {
            const message = isJSON && payload && payload.error
                ? payload.error
                : `HTTP ${response.status}: ${response.statusText}`;
            throw new Error(message);
        }

        return payload;
    },

    async getStatus() {
        return this.request('/status');
    },

    async setFrequency(frequency) {
        return this.request('/frequency', {
            method: 'POST',
            body: JSON.stringify({ frequency })
        });
    },

    async play(payload = {}) {
        return this.request('/playback/play', {
            method: 'POST',
            body: JSON.stringify(payload)
        });
    },

    async pause() {
        return this.request('/playback/pause', { method: 'POST' });
    },

    async stopPlayback() {
        return this.request('/playback/stop', { method: 'POST' });
    },

    async next() {
        return this.request('/playback/next', { method: 'POST' });
    },

    async prev() {
        return this.request('/playback/prev', { method: 'POST' });
    },

    async uploadFile(file) {
        const formData = new FormData();
        formData.append('file', file);

        return this.request('/files/upload', {
            method: 'POST',
            body: formData,
            headers: {}
        });
    },

    async getFiles() {
        return this.request('/files');
    },

    async deleteFile(fileID) {
        return this.request(`/files/${encodeURIComponent(fileID)}`, {
            method: 'DELETE'
        });
    },

    async getPlaylist() {
        return this.request('/playlist');
    },

    async addToPlaylist(fileID, filename) {
        return this.request('/playlist/add', {
            method: 'POST',
            body: JSON.stringify({ file_id: fileID, filename })
        });
    },

    async removeFromPlaylist(fileID) {
        return this.request(`/playlist/${encodeURIComponent(fileID)}`, {
            method: 'DELETE'
        });
    },

    async reorderPlaylist(fromIndex, toIndex) {
        return this.request('/playlist/reorder', {
            method: 'POST',
            body: JSON.stringify({ from_index: fromIndex, to_index: toIndex })
        });
    },

    async playTrack(index) {
        return this.play({ index });
    },

    async playFile(fileID) {
        return this.play({ file_id: fileID });
    }
};

window.API = API;
