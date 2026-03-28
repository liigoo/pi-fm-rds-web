package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/liigoo/pi-fm-rds-go/internal/audio"
	"github.com/liigoo/pi-fm-rds-go/internal/config"
	"github.com/liigoo/pi-fm-rds-go/internal/playlist"
	"github.com/liigoo/pi-fm-rds-go/internal/process"
	"github.com/liigoo/pi-fm-rds-go/internal/storage"
	"github.com/liigoo/pi-fm-rds-go/internal/websocket"
)

// Managers 所有管理器的集合
type Managers struct {
	audio    *audio.Manager
	storage  storage.Manager
	process  process.Manager
	playlist playlist.Manager
	wsHub    *websocket.Hub
}

func main() {
	// 1. 解析命令行参数
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	// 2. 加载配置
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("Configuration loaded successfully")
	log.Printf("  Server: %s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("  FM Frequency: %.1f MHz", cfg.PiFmRds.DefaultFrequency)
	log.Printf("  Upload Dir: %s", cfg.Storage.UploadDir)
	log.Printf("  Sample Rate: %d Hz", cfg.Audio.SampleRate)
	log.Printf("  Max WebSocket Clients: %d", cfg.WebSocket.MaxClients)

	// 3. 初始化所有管理器
	managers, err := initializeManagers(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize managers: %v", err)
	}
	log.Printf("All managers initialized successfully")

	// 4. 启动依赖检查（FR-008, FR-009）
	if err := performStartupChecks(managers, cfg); err != nil {
		log.Printf("Warning: Startup checks failed: %v", err)
	}

	// 5. 启动 WebSocket Hub
	go managers.wsHub.Run()
	log.Printf("WebSocket hub started")

	// 6. 配置路由
	mux := setupRoutes(managers, cfg)

	// 7. 创建 HTTP 服务器
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 8. 启动服务器（在 goroutine 中）
	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("Starting HTTP server on %s", addr)
		serverErrors <- server.ListenAndServe()
	}()

	// 9. 等待中断信号
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		log.Fatalf("Server error: %v", err)

	case sig := <-shutdown:
		log.Printf("Received signal: %v. Starting graceful shutdown...", sig)

		// 10. 优雅关闭
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// 关闭 HTTP 服务器
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("Error during server shutdown: %v", err)
			server.Close()
		}

		// 关闭所有管理器
		shutdownManagers(managers)

		log.Printf("Server stopped gracefully")
	}
}

// initializeManagers 初始化所有管理器
func initializeManagers(cfg *config.Config) (*Managers, error) {
	// 创建存储目录
	if err := os.MkdirAll(cfg.Storage.UploadDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create upload dir: %w", err)
	}
	if err := os.MkdirAll(cfg.Storage.TranscodedDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create transcoded dir: %w", err)
	}

	// 初始化音频管理器
	audioMgr := audio.NewManager(&audio.Config{
		SampleRate: cfg.Audio.SampleRate,
		Channels:   cfg.Audio.Channels,
	})

	// 初始化存储管理器
	storageMgr := storage.NewManager(
		cfg.Storage.UploadDir,
		cfg.Storage.TranscodedDir,
		cfg.Storage.MaxFileSize,
		cfg.Storage.MaxTotalSize,
	)

	// 初始化进程管理器
	processMgr := process.NewManager(cfg.PiFmRds.BinaryPath)

	// 初始化播放列表管理器
	playlistMgr := playlist.NewManager()

	// 初始化 WebSocket Hub
	wsHub := websocket.NewHub(cfg.WebSocket.MaxClients)

	return &Managers{
		audio:    audioMgr,
		storage:  storageMgr,
		process:  processMgr,
		playlist: playlistMgr,
		wsHub:    wsHub,
	}, nil
}

// performStartupChecks 执行启动检查（FR-008, FR-009）
func performStartupChecks(managers *Managers, cfg *config.Config) error {
	log.Printf("Performing startup checks...")

	// FR-008: 检查 pi_fm_rds 二进制文件
	if _, err := os.Stat(cfg.PiFmRds.BinaryPath); os.IsNotExist(err) {
		return fmt.Errorf("pi_fm_rds binary not found at %s", cfg.PiFmRds.BinaryPath)
	}
	log.Printf("✓ pi_fm_rds binary found")

	// FR-009: 验证 GPIO 4 可用性（仅在 Linux 上）
	if _, err := os.Stat("/sys/class/gpio"); err == nil {
		if err := managers.process.ValidateGPIO(); err != nil {
			log.Printf("⚠ GPIO validation warning: %v", err)
		} else {
			log.Printf("✓ GPIO 4 validated")
		}
	} else {
		log.Printf("⚠ Not running on Linux, skipping GPIO validation")
	}

	// 清理孤儿进程
	if err := managers.process.CleanupOrphans(); err != nil {
		log.Printf("⚠ Failed to cleanup orphan processes: %v", err)
	} else {
		log.Printf("✓ Orphan processes cleaned up")
	}

	log.Printf("Startup checks completed")
	return nil
}

// setupRoutes 配置路由
func setupRoutes(managers *Managers, cfg *config.Config) *http.ServeMux {
	mux := http.NewServeMux()

	// API 路由（将在后续任务中实现）
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// WebSocket 路由（将在后续任务中实现）
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		// WebSocket 升级逻辑将在后续任务中实现
		http.Error(w, "WebSocket endpoint not yet implemented", http.StatusNotImplemented)
	})

	// 静态文件服务（将在后续任务中实现）
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("<html><body><h1>Pi FM RDS Server</h1><p>Server is running</p></body></html>"))
			return
		}
		http.NotFound(w, r)
	})

	log.Printf("Routes configured")
	return mux
}

// shutdownManagers 关闭所有管理器
func shutdownManagers(managers *Managers) {
	log.Printf("Shutting down managers...")

	// 停止 WebSocket Hub
	managers.wsHub.Stop()
	log.Printf("✓ WebSocket hub stopped")

	// 停止音频管理器
	if err := managers.audio.Stop(); err != nil {
		log.Printf("⚠ Error stopping audio manager: %v", err)
	} else {
		log.Printf("✓ Audio manager stopped")
	}

	// 停止进程管理器
	if managers.process.IsRunning() {
		if err := managers.process.Stop(); err != nil {
			log.Printf("⚠ Error stopping process manager: %v", err)
		} else {
			log.Printf("✓ Process manager stopped")
		}
	}

	log.Printf("All managers shut down")
}
