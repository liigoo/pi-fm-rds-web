// 复古频谱可视化
const Spectrum = {
    init() {
        this.canvas = document.getElementById('spectrumCanvas');
        if (!this.canvas) return;

        this.ctx = this.canvas.getContext('2d');
        this.smooth = new Array(64).fill(0);
        this.peaks = new Array(64).fill(0);
        this.phase = 0;

        this.resizeCanvas();
        window.addEventListener('resize', () => this.resizeCanvas());
        this.startAnimation();
    },

    resizeCanvas() {
        const rect = this.canvas.getBoundingClientRect();
        this.canvas.width = Math.max(320, Math.floor(rect.width));
        this.canvas.height = Math.max(220, Math.floor(rect.height));
    },

    startAnimation() {
        const drawFrame = () => {
            const state = AppState.getState();
            const bins = this.computeBins(state);
            this.draw(bins, state);
            requestAnimationFrame(drawFrame);
        };
        requestAnimationFrame(drawFrame);
    },

    computeBins(state) {
        const targetCount = this.smooth.length;
        const raw = Array.isArray(state.spectrumData) ? state.spectrumData : [];

        if (raw.length === 0 && state.isRunning) {
            this.phase += 0.05;
            return this.smooth.map((_, i) => {
                const wave = Math.sin(this.phase + i * 0.35) * 0.26 + 0.32;
                const noise = Math.random() * 0.16;
                return Math.max(0, Math.min(1, wave + noise));
            });
        }

        if (raw.length === 0) {
            return new Array(targetCount).fill(0);
        }

        const max = Math.max(...raw, 1);
        const step = raw.length / targetCount;
        const bins = new Array(targetCount).fill(0);

        for (let i = 0; i < targetCount; i += 1) {
            const start = Math.floor(i * step);
            const end = Math.min(raw.length, Math.max(start + 1, Math.floor((i + 1) * step)));
            let localMax = 0;
            for (let j = start; j < end; j += 1) {
                localMax = Math.max(localMax, raw[j] || 0);
            }
            bins[i] = Math.min(1, localMax / max);
        }

        return bins;
    },

    draw(bins, state) {
        const ctx = this.ctx;
        const { width, height } = this.canvas;

        ctx.clearRect(0, 0, width, height);

        const bg = ctx.createLinearGradient(0, 0, 0, height);
        bg.addColorStop(0, '#111111');
        bg.addColorStop(1, '#060606');
        ctx.fillStyle = bg;
        ctx.fillRect(0, 0, width, height);

        this.drawGrid(width, height);

        const marginX = 14;
        const marginY = 14;
        const usableWidth = width - marginX * 2;
        const usableHeight = height - marginY * 2;
        const barWidth = usableWidth / bins.length;

        for (let i = 0; i < bins.length; i += 1) {
            const value = bins[i];
            const smoothed = this.smooth[i] * 0.74 + value * 0.26;
            this.smooth[i] = smoothed;

            const peakDecay = state.isRunning ? 0.008 : 0.02;
            this.peaks[i] = Math.max(smoothed, this.peaks[i] - peakDecay);

            const x = marginX + i * barWidth;
            const h = smoothed * usableHeight;
            const y = height - marginY - h;

            ctx.fillStyle = '#d7d7d7';
            ctx.fillRect(x + 1, y, Math.max(1, barWidth - 2), h);

            const peakY = height - marginY - this.peaks[i] * usableHeight;
            ctx.fillStyle = '#b41d1d';
            ctx.fillRect(x + 1, peakY, Math.max(1, barWidth - 2), 2);
        }

        if (!state.isRunning && !state.isPaused) {
            ctx.fillStyle = 'rgba(236, 236, 236, 0.72)';
            ctx.font = "14px 'IBM Plex Mono', monospace";
            ctx.textAlign = 'center';
            ctx.fillText('STANDBY', width / 2, height / 2);
        }
    },

    drawGrid(width, height) {
        const ctx = this.ctx;
        ctx.strokeStyle = 'rgba(196, 196, 196, 0.12)';
        ctx.lineWidth = 1;

        for (let i = 1; i <= 5; i += 1) {
            const y = (height / 6) * i;
            ctx.beginPath();
            ctx.moveTo(0, y);
            ctx.lineTo(width, y);
            ctx.stroke();
        }

        for (let i = 1; i <= 8; i += 1) {
            const x = (width / 9) * i;
            ctx.beginPath();
            ctx.moveTo(x, 0);
            ctx.lineTo(x, height);
            ctx.stroke();
        }
    }
};

window.Spectrum = Spectrum;
