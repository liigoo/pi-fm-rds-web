// 控制面板
const Controls = {
    init() {
        this.frequencySlider = document.getElementById('frequencySlider');
        this.frequencyInput = document.getElementById('frequencyInput');
        this.frequencyValue = document.getElementById('frequencyValue');
        this.startBtn = document.getElementById('startBtn');
        this.stopBtn = document.getElementById('stopBtn');
        this.uploadInput = document.getElementById('uploadInput');
        this.uploadBtn = document.getElementById('uploadBtn');
        this.statusText = document.getElementById('statusText');

        this.bindEvents();
        this.updateUI(AppState.getState());

        AppState.subscribe((state) => this.updateUI(state));
    },

    bindEvents() {
        // 频率滑块
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

        // 频率输入框
        this.frequencyInput?.addEventListener('change', async (e) => {
            let freq = parseFloat(e.target.value);
            freq = Math.max(87.5, Math.min(108.0, freq));
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

        // 启动按钮
        this.startBtn?.addEventListener('click', async () => {
            try {
                await API.start();
                AppState.setState({ isRunning: true });
                setTimeout(async () => {
                    try {
                        const status = await API.getStatus();
                        AppState.setState({ isRunning: status.running || false, frequency: status.frequency || 88.0 });
                    } catch (_) {}
                }, 800);
            } catch (err) {
                alert('启动失败: ' + err.message);
            }
        });

        // 停止按钮
        this.stopBtn?.addEventListener('click', async () => {
            try {
                await API.stop();
            } catch (_) {}
            AppState.setState({ isRunning: false });
        });

        // 文件上传
        this.uploadBtn?.addEventListener('click', () => {
            this.uploadInput?.click();
        });

        this.uploadInput?.addEventListener('change', async (e) => {
            const file = e.target.files[0];
            if (!file) return;

            try {
                await API.uploadFile(file);
                alert('上传成功');
                e.target.value = '';
                const playlist = await API.getPlaylist();
                AppState.setState({ playlist: (playlist && playlist.items) || [] });
            } catch (err) {
                alert('上传失败: ' + err.message);
            }
        });
    },

    updateUI(state) {
        if (this.frequencyValue) {
            this.frequencyValue.textContent = state.frequency.toFixed(1);
        }
        if (this.frequencySlider) {
            this.frequencySlider.value = state.frequency;
        }
        if (this.frequencyInput) {
            this.frequencyInput.value = state.frequency.toFixed(1);
        }

        if (this.statusText) {
            this.statusText.textContent = state.isRunning ? '运行中' : '已停止';
            this.statusText.className = state.isRunning ? 'status-running' : 'status-stopped';
        }

        if (this.startBtn) {
            this.startBtn.disabled = state.isRunning;
        }
        if (this.stopBtn) {
            this.stopBtn.disabled = !state.isRunning;
        }
    }
};

window.Controls = Controls;
