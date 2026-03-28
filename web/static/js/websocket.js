// WebSocket 客户端
class WSClient {
    constructor() {
        this.ws = null;
        this.reconnectDelay = 1000;
        this.maxReconnectDelay = 30000;
        this.reconnectAttempts = 0;
        this.messageHandlers = [];
    }

    connect() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsURL = `${protocol}//${window.location.host}/ws`;

        this.ws = new WebSocket(wsURL);

        this.ws.onopen = () => {
            console.log('WebSocket connected');
            this.reconnectAttempts = 0;
            this.reconnectDelay = 1000;
            AppState.setState({ connectionStatus: 'connected' });
        };

        this.ws.onmessage = (event) => {
            try {
                const message = JSON.parse(event.data);
                this.handleMessage(message);
            } catch (err) {
                console.error('WebSocket message error:', err);
            }
        };

        this.ws.onerror = (err) => {
            console.error('WebSocket error:', err);
        };

        this.ws.onclose = () => {
            console.log('WebSocket disconnected');
            AppState.setState({ connectionStatus: 'disconnected' });
            this.scheduleReconnect();
        };
    }

    handleMessage(message) {
        const { type, data } = message;

        switch (type) {
            case 'status':
                AppState.setState({
                    isRunning: data.running,
                    frequency: data.frequency
                });
                break;
            case 'spectrum':
                AppState.setState({ spectrumData: data });
                break;
            case 'playlist':
                AppState.setState({ playlist: data });
                break;
            case 'error':
                console.error('Server error:', data);
                break;
        }

        this.messageHandlers.forEach(handler => handler(message));
    }

    scheduleReconnect() {
        this.reconnectAttempts++;
        const delay = Math.min(
            this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1),
            this.maxReconnectDelay
        );

        console.log(`Reconnecting in ${delay}ms...`);
        setTimeout(() => this.connect(), delay);
    }

    onMessage(handler) {
        this.messageHandlers.push(handler);
    }

    disconnect() {
        if (this.ws) {
            this.ws.close();
            this.ws = null;
        }
    }
}

window.WSClient = WSClient;
