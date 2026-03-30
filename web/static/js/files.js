// 文件列表视图
const Files = {
    init() {
        this.body = document.getElementById('fileListBody');
        this.updateUI(AppState.getState());
        AppState.subscribe((state) => this.updateUI(state));
    },

    updateUI(state) {
        if (!this.body) return;

        const files = Array.isArray(state.files) ? state.files : [];
        if (files.length === 0) {
            this.body.innerHTML = `
                <tr>
                    <td colspan="3" class="empty-row">暂无文件</td>
                </tr>
            `;
            return;
        }

        this.body.innerHTML = files.map((file) => {
            const selected = file.ID === state.selectedFileID;
            const filename = this.escapeHtml(file.Filename || '未知文件');
            const sizeText = this.formatSize(file.Size || 0);

            return `
                <tr class="file-row ${selected ? 'selected' : ''}" data-id="${file.ID}">
                    <td title="${filename}">${filename}</td>
                    <td>${sizeText}</td>
                    <td>
                        <div class="file-actions">
                            <button class="btn btn-queue" data-action="queue" data-id="${file.ID}" data-name="${filename}" title="加入队列">＋</button>
                            <button class="btn btn-danger" data-action="delete" data-id="${file.ID}">删除</button>
                        </div>
                    </td>
                </tr>
            `;
        }).join('');

        this.bindEvents();
    },

    bindEvents() {
        this.body.querySelectorAll('.file-row').forEach((row) => {
            row.addEventListener('click', () => {
                AppState.setState({ selectedFileID: row.dataset.id || '' });
            });
        });

        this.body.querySelectorAll('button[data-action]').forEach((btn) => {
            btn.addEventListener('click', async (e) => {
                e.stopPropagation();

                const action = btn.dataset.action;
                const fileID = btn.dataset.id;

                try {
                    if (action === 'queue') {
                        await API.addToPlaylist(fileID, btn.dataset.name || '未命名文件');
                    } else if (action === 'delete') {
                        if (!confirm('确认删除该文件？')) return;
                        await API.deleteFile(fileID);
                        const state = AppState.getState();
                        if (state.selectedFileID === fileID) {
                            AppState.setState({ selectedFileID: '' });
                        }
                    }

                    await App.loadSnapshot();
                } catch (err) {
                    const actionMap = { queue: '加入队列', delete: '删除' };
                    alert(`${actionMap[action] || '操作'}失败: ${err.message}`);
                }
            });
        });
    },

    formatSize(size) {
        const n = Number(size) || 0;
        if (n < 1024) return `${n} B`;
        if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
        return `${(n / (1024 * 1024)).toFixed(1)} MB`;
    },

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
};

window.Files = Files;
