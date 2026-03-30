// 主控制面板
const Controls = {
    FREQ_MIN: 87.5,
    FREQ_MAX: 108.0,
    FREQ_STEP: 0.1,

    init() {
        this.frequencySlider = document.getElementById('frequencySlider');
        this.frequencyInput = document.getElementById('frequencyInput');
        this.frequencyValue = document.getElementById('frequencyValue');
        this.freqKnob = document.getElementById('freqKnob');

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

        this.knobDragging = false;
        this.knobStartY = 0;
        this.knobStartFrequency = 88.0;

        this.bindEvents();
        this.updateUI(AppState.getState());
        AppState.subscribe((state) => this.updateUI(state));
    },

    clampFrequency(freq) {
        const n = Number(freq);
        if (!Number.isFinite(n)) return 88.0;
        const bounded = Math.max(this.FREQ_MIN, Math.min(this.FREQ_MAX, n));
        return Math.round(bounded / this.FREQ_STEP) * this.FREQ_STEP;
    },

    frequencyToAngle(freq) {
        const ratio = (freq - this.FREQ_MIN) / (this.FREQ_MAX - this.FREQ_MIN);
        return -140 + ratio * 280;
    },

    setKnobRotation(freq) {
        if (!this.freqKnob) return;
        const angle = this.frequencyToAngle(freq);
        this.freqKnob.style.setProperty('--knob-angle', `${angle}deg`);
    },

    previewFrequency(freq) {
        const normalized = this.clampFrequency(freq);
        AppState.setState({ frequency: normalized });
        return normalized;
    },

    async commitFrequency(freq) {
        const normalized = this.clampFrequency(freq);
        await API.setFrequency(normalized);
        AppState.setState({ frequency: normalized });
        return normalized;
    },

    bindKnobEvents() {
        if (!this.freqKnob) return;

        this.freqKnob.addEventListener('mousedown', (e) => {
            e.preventDefault();
            this.knobDragging = true;
            this.knobStartY = e.clientY;
            this.knobStartFrequency = this.clampFrequency(AppState.getState().frequency);
            this.freqKnob.classList.add('dragging');
        });

        window.addEventListener('mousemove', (e) => {
            if (!this.knobDragging) return;
            const delta = this.knobStartY - e.clientY;
            const nextFrequency = this.knobStartFrequency + delta * 0.02;
            this.previewFrequency(nextFrequency);
        });

        window.addEventListener('mouseup', async () => {
            if (!this.knobDragging) return;
            this.knobDragging = false;
            this.freqKnob.classList.remove('dragging');
            const nextFrequency = this.clampFrequency(AppState.getState().frequency);
            try {
                await this.commitFrequency(nextFrequency);
            } catch (err) {
                alert('设置频率失败: ' + err.message);
            }
        });

        this.freqKnob.addEventListener('wheel', async (e) => {
            e.preventDefault();
            const current = this.clampFrequency(AppState.getState().frequency);
            const direction = e.deltaY < 0 ? 1 : -1;
            const nextFrequency = current + direction * this.FREQ_STEP;
            const normalized = this.previewFrequency(nextFrequency);

            try {
                await this.commitFrequency(normalized);
            } catch (err) {
                alert('设置频率失败: ' + err.message);
            }
        }, { passive: false });
    },

    bindEvents() {
        this.frequencySlider?.addEventListener('input', (e) => {
            const freq = this.clampFrequency(parseFloat(e.target.value));
            this.previewFrequency(freq);
        });

        this.frequencySlider?.addEventListener('change', async (e) => {
            const freq = this.clampFrequency(parseFloat(e.target.value));
            try {
                await this.commitFrequency(freq);
            } catch (err) {
                alert('设置频率失败: ' + err.message);
            }
        });

        this.frequencyInput?.addEventListener('change', async (e) => {
            const freq = this.clampFrequency(parseFloat(e.target.value));
            this.previewFrequency(freq);

            try {
                await this.commitFrequency(freq);
            } catch (err) {
                alert('设置频率失败: ' + err.message);
            }
        });

        this.bindKnobEvents();

        this.playBtn?.addEventListener('click', async () => {
            const state = AppState.getState();
            try {
                if (state.selectedFileID) {
                    await API.playFile(state.selectedFileID);
                } else {
                    await API.play();
                }
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
        const freq = this.clampFrequency(state.frequency);

        if (this.frequencyValue) {
            this.frequencyValue.textContent = freq.toFixed(1);
        }
        if (this.frequencySlider) {
            this.frequencySlider.value = String(freq);
        }
        if (this.frequencyInput) {
            this.frequencyInput.value = freq.toFixed(1);
        }

        this.setKnobRotation(freq);

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
        const hasSelection = !!state.selectedFileID;
        if (this.playBtn) this.playBtn.disabled = state.isRunning || (!hasPlaylist && !hasSelection);
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
