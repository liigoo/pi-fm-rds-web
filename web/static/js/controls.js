// 主控制面板
const Controls = {
    init() {
        this.frequencySlider = document.getElementById('frequencySlider');
        this.frequencyInput = document.getElementById('frequencyInput');
        this.frequencyValue = document.getElementById('frequencyValue');

        this.prevBtn = document.getElementById('prevBtn');
        this.playBtn = document.getElementById('playBtn');
        this.pauseBtn = document.getElementById('pauseBtn');
        this.nextBtn = document.getElementById('nextBtn');
        this.stopBtn = document.getElementById('stopBtn');

        this.uploadInput = document.getElementById('uploadInput');
        this.uploadBtn = document.getElementById('uploadBtn');
        this.deleteSelectedBtn = document.getElementById('deleteSelectedBtn');

        this.statusText = document.getElementById('statusText');
        this.playStateBadge = document.getElementById('playStateBadge');
        this.currentTrackName = document.getElementById('currentTrackName');
        this.connectionStatus = document.getElementById('connectionStatus');
        this.connectionDot = this.connectionStatus?.querySelector('.status-dot') || null;
        this.connectionText = this.connectionStatus?.querySelector('.status-text') || null;

        this.bindEvents();
        this.updateUI(AppState.getState());
        AppState.subscribe((state) => this.updateUI(state));
    },

    bindEvents() {
        this.frequencySlider?.addEventListener('input', (e) => {
            const freq = parseFloat(e.target.value);
            this.frequencyValue.textContent = freq.toFixed(1);
            if (this.frequencyInput) {
                this.frequencyInput.value = freq.toFixed(1);
            }
        });

        this.frequencySlider?.addEventListener('change', async (e) => {
            const freq = parseFloat(e.target.value);
            try {
                await API.setFrequency(freq);
                AppState.setState({ frequency: freq });
            } catch (err) {
                alert('设置频率失败: ' + err.message);
            }
        });

        this.frequencyInput?.addEventListener('change', async (e) => {
            let freq = parseFloat(e.target.value);
            freq = Math.max(87.5, Math.min(108.0, isNaN(freq) ? 88.0 : freq));
            e.target.value = freq.toFixed(1);

            if (this.frequencySlider) {
                this.frequencySlider.value = freq;
            }
            this.frequencyValue.textContent = freq.toFixed(1);

            try {
                await API.setFrequency(freq);
                AppState.setState({ frequency: freq });
            } catch (err) {
                alert('设置频率失败: ' + err.message);
            }
        });

        this.playBtn?.addEventListener('click', async () => {
            try {
                await API.play();
                await App.loadSnapshot();
            } catch (err) {
                alert('播放失败: ' + err.message);
            }
        });

        this.pauseBtn?.addEventListener('click', async () => {
            try {
                await API.pause();
                await App.loadSnapshot();
            } catch (err) {
                alert('暂停失败: ' + err.message);
            }
        });

        this.stopBtn?.addEventListener('click', async () => {
            try {
                await API.stopPlayback();
                await App.loadSnapshot();
            } catch (err) {
                alert('停止失败: ' + err.message);
            }
        });

        this.nextBtn?.addEventListener('click', async () => {
            try {
                await API.next();
                await App.loadSnapshot();
            } catch (err) {
                alert('下一曲失败: ' + err.message);
            }
        });

        this.prevBtn?.addEventListener('click', async () => {
            try {
                await API.prev();
                await App.loadSnapshot();
            } catch (err) {
                alert('上一曲失败: ' + err.message);
            }
        });

        this.uploadBtn?.addEventListener('click', () => {
            this.uploadInput?.click();
        });

        this.uploadInput?.addEventListener('change', async (e) => {
            const file = e.target.files?.[0];
            if (!file) return;

            try {
                const uploaded = await API.uploadFile(file);
                if (uploaded && uploaded.file_id) {
                    await API.addToPlaylist(uploaded.file_id, file.name);
                }
                e.target.value = '';
                await App.loadSnapshot();
            } catch (err) {
                alert('上传失败: ' + err.message);
            }
        });

        this.deleteSelectedBtn?.addEventListener('click', async () => {
            const state = AppState.getState();
            if (!state.selectedFileID) {
                alert('请先在文件列表选择一个文件');
                return;
            }

            if (!confirm('确认删除所选文件？')) return;

            try {
                await API.deleteFile(state.selectedFileID);
                AppState.setState({ selectedFileID: '' });
                await App.loadSnapshot();
            } catch (err) {
                alert('删除失败: ' + err.message);
            }
        });
    },

    updateUI(state) {
        const freq = Number.isFinite(state.frequency) ? state.frequency : 88.0;

        if (this.frequencyValue) {
            this.frequencyValue.textContent = freq.toFixed(1);
        }
        if (this.frequencySlider) {
            this.frequencySlider.value = String(freq);
        }
        if (this.frequencyInput) {
            this.frequencyInput.value = freq.toFixed(1);
        }

        const statusClass = state.isRunning ? 'status-running' : (state.isPaused ? 'status-paused' : 'status-stopped');
        const statusText = state.isRunning ? '播放中' : (state.isPaused ? '已暂停' : '已停止');

        if (this.statusText) {
            this.statusText.textContent = statusText;
            this.statusText.className = statusClass;
        }

        if (this.playStateBadge) {
            this.playStateBadge.textContent = statusText;
            this.playStateBadge.className = `state-badge ${state.isRunning ? 'playing' : (state.isPaused ? 'paused' : 'stopped')}`;
        }

        if (this.currentTrackName) {
            this.currentTrackName.textContent = this.getCurrentTrackName(state);
        }

        this.updateConnection(state.connectionStatus);

        const hasPlaylist = Array.isArray(state.playlist) && state.playlist.length > 0;
        if (this.playBtn) this.playBtn.disabled = state.isRunning || !hasPlaylist;
        if (this.pauseBtn) this.pauseBtn.disabled = !state.isRunning;
        if (this.stopBtn) this.stopBtn.disabled = !state.isRunning && !state.isPaused;
        if (this.nextBtn) this.nextBtn.disabled = !hasPlaylist;
        if (this.prevBtn) this.prevBtn.disabled = !hasPlaylist;
    },

    updateConnection(status) {
        if (!this.connectionDot || !this.connectionText) return;

        const connected = status === 'connected';
        this.connectionDot.classList.remove('connected', 'disconnected');
        this.connectionDot.classList.add(connected ? 'connected' : 'disconnected');
        this.connectionText.textContent = connected ? '已连接' : '已断开';
    },

    getCurrentTrackName(state) {
        const current = (state.playlist || []).find((item) => item.FileID === state.currentFileID);
        if (current && current.Filename) return current.Filename;
        if (!state.isRunning && !state.isPaused) return '--';
        return '已就绪';
    }
};

window.Controls = Controls;
