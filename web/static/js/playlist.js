// 播放列表
const Playlist = {
    init() {
        this.container = document.getElementById('playlistContainer');
        this.updateUI(AppState.getState());
        AppState.subscribe((state) => this.updateUI(state));
    },

    updateUI(state) {
        if (!this.container) return;

        if (state.playlist.length === 0) {
            this.container.innerHTML = '<div class="empty-playlist">暂无曲目</div>';
            return;
        }

        this.container.innerHTML = state.playlist.map((track, index) => `
            <div class="playlist-item ${index === state.currentTrack ? 'active' : ''}"
                 data-index="${index}"
                 draggable="true">
                <div class="track-number">${index + 1}</div>
                <div class="track-info">
                    <div class="track-name">${this.escapeHtml(track.Filename || track.name || '')}</div>
                    <div class="track-duration">${this.formatDuration(track.Duration || track.duration)}</div>
                </div>
                <div class="track-actions">
                    <button class="btn-play" data-index="${index}">▶</button>
                </div>
            </div>
        `).join('');

        this.bindEvents();
    },

    bindEvents() {
        // 播放按钮
        this.container.querySelectorAll('.btn-play').forEach(btn => {
            btn.addEventListener('click', async (e) => {
                const index = parseInt(e.target.dataset.index);
                try {
                    await API.playTrack(index);
                    AppState.setState({ currentTrack: index });
                } catch (err) {
                    alert('播放失败: ' + err.message);
                }
            });
        });

        // 拖拽排序
        const items = this.container.querySelectorAll('.playlist-item');
        items.forEach(item => {
            item.addEventListener('dragstart', (e) => {
                e.dataTransfer.effectAllowed = 'move';
                e.dataTransfer.setData('text/plain', e.target.dataset.index);
                e.target.classList.add('dragging');
            });

            item.addEventListener('dragend', (e) => {
                e.target.classList.remove('dragging');
            });

            item.addEventListener('dragover', (e) => {
                e.preventDefault();
                e.dataTransfer.dropEffect = 'move';
            });

            item.addEventListener('drop', (e) => {
                e.preventDefault();
                const fromIndex = parseInt(e.dataTransfer.getData('text/plain'));
                const toIndex = parseInt(e.currentTarget.dataset.index);

                if (fromIndex !== toIndex) {
                    this.reorderPlaylist(fromIndex, toIndex);
                }
            });
        });
    },

    reorderPlaylist(fromIndex, toIndex) {
        const state = AppState.getState();
        const playlist = [...state.playlist];
        const [item] = playlist.splice(fromIndex, 1);
        playlist.splice(toIndex, 0, item);
        AppState.setState({ playlist });
    },

    formatDuration(seconds) {
        if (!seconds) return '--:--';
        const mins = Math.floor(seconds / 60);
        const secs = Math.floor(seconds % 60);
        return `${mins}:${secs.toString().padStart(2, '0')}`;
    },

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
};

window.Playlist = Playlist;
