// 播放队列视图
const Playlist = {
    init() {
        this.container = document.getElementById('playlistContainer');
        this.updateUI(AppState.getState());
        AppState.subscribe((state) => this.updateUI(state));
    },

    updateUI(state) {
        if (!this.container) return;

        const list = Array.isArray(state.playlist) ? state.playlist : [];
        if (list.length === 0) {
            this.container.innerHTML = '<div class="empty-row">队列为空，上传后可自动加入</div>';
            return;
        }

        this.container.innerHTML = list.map((track, index) => {
            const active = track.FileID === state.currentFileID || index === state.currentTrack;
            const name = this.escapeHtml(track.Filename || track.filename || `Track ${index + 1}`);
            return `
                <div class="playlist-item ${active ? 'active' : ''}" data-index="${index}" draggable="true">
                    <div class="track-number">${index + 1}</div>
                    <div class="track-info">
                        <div class="track-name" title="${name}">${name}</div>
                        <div class="track-duration">${this.formatDuration(track.Duration || track.duration)}</div>
                    </div>
                    <div class="track-actions">
                        <button class="btn btn-play" data-action="play" data-index="${index}">播放</button>
                        <button class="btn btn-danger" data-action="remove" data-file-id="${track.FileID}">移除</button>
                    </div>
                </div>
            `;
        }).join('');

        this.bindEvents();
    },

    bindEvents() {
        this.container.querySelectorAll('button[data-action="play"]').forEach((btn) => {
            btn.addEventListener('click', async (e) => {
                e.stopPropagation();
                const index = parseInt(e.currentTarget.dataset.index, 10);
                try {
                    await API.playTrack(index);
                    await App.loadSnapshot();
                } catch (err) {
                    alert('播放失败: ' + err.message);
                }
            });
        });

        this.container.querySelectorAll('button[data-action="remove"]').forEach((btn) => {
            btn.addEventListener('click', async (e) => {
                e.stopPropagation();
                const fileID = e.currentTarget.dataset.fileId;
                try {
                    await API.removeFromPlaylist(fileID);
                    await App.loadSnapshot();
                } catch (err) {
                    alert('移除失败: ' + err.message);
                }
            });
        });

        const items = this.container.querySelectorAll('.playlist-item');
        items.forEach((item) => {
            item.addEventListener('dragstart', (e) => {
                e.dataTransfer.effectAllowed = 'move';
                e.dataTransfer.setData('text/plain', e.currentTarget.dataset.index);
                e.currentTarget.classList.add('dragging');
            });

            item.addEventListener('dragend', (e) => {
                e.currentTarget.classList.remove('dragging');
            });

            item.addEventListener('dragover', (e) => {
                e.preventDefault();
                e.dataTransfer.dropEffect = 'move';
            });

            item.addEventListener('drop', async (e) => {
                e.preventDefault();
                const fromIndex = parseInt(e.dataTransfer.getData('text/plain'), 10);
                const toIndex = parseInt(e.currentTarget.dataset.index, 10);
                if (Number.isNaN(fromIndex) || Number.isNaN(toIndex) || fromIndex === toIndex) {
                    return;
                }

                try {
                    await API.reorderPlaylist(fromIndex, toIndex);
                    await App.loadSnapshot();
                } catch (err) {
                    alert('排序失败: ' + err.message);
                }
            });
        });
    },

    formatDuration(duration) {
        if (!duration) return '--:--';
        const seconds = typeof duration === 'number' ? duration : Number(duration);
        if (!Number.isFinite(seconds) || seconds <= 0) return '--:--';

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
