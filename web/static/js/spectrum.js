// 频谱可视化
const Spectrum = {
    init() {
        this.canvas = document.getElementById('spectrumCanvas');
        if (!this.canvas) return;

        this.ctx = this.canvas.getContext('2d');
        this.resizeCanvas();

        window.addEventListener('resize', () => this.resizeCanvas());
        AppState.subscribe((state) => {
            if (state.spectrumData && state.spectrumData.length > 0) {
                this.draw(state.spectrumData);
            }
        });

        this.startAnimation();
    },

    resizeCanvas() {
        const container = this.canvas.parentElement;
        this.canvas.width = container.clientWidth;
        this.canvas.height = container.clientHeight || 200;
    },

    startAnimation() {
        let lastTime = 0;
        const fps = 15;
        const interval = 1000 / fps;

        const animate = (currentTime) => {
            requestAnimationFrame(animate);

            const elapsed = currentTime - lastTime;
            if (elapsed < interval) return;

            lastTime = currentTime - (elapsed % interval);

            const state = AppState.getState();
            if (state.spectrumData && state.spectrumData.length > 0) {
                this.draw(state.spectrumData);
            } else {
                this.drawEmpty();
            }
        };

        requestAnimationFrame(animate);
    },

    draw(data) {
        const { width, height } = this.canvas;
        const ctx = this.ctx;

        ctx.clearRect(0, 0, width, height);

        const barCount = Math.min(data.length, 64);
        const barWidth = width / barCount;
        const maxValue = Math.max(...data, 1);

        for (let i = 0; i < barCount; i++) {
            const value = data[i] || 0;
            const barHeight = (value / maxValue) * height * 0.9;
            const x = i * barWidth;
            const y = height - barHeight;

            const hue = (i / barCount) * 120 + 200;
            ctx.fillStyle = `hsl(${hue}, 70%, 60%)`;
            ctx.fillRect(x, y, barWidth - 2, barHeight);
        }
    },

    drawEmpty() {
        const { width, height } = this.canvas;
        const ctx = this.ctx;

        ctx.clearRect(0, 0, width, height);
        ctx.fillStyle = '#666';
        ctx.font = '16px sans-serif';
        ctx.textAlign = 'center';
        ctx.textBaseline = 'middle';
        ctx.fillText('等待音频数据...', width / 2, height / 2);
    }
};

window.Spectrum = Spectrum;
