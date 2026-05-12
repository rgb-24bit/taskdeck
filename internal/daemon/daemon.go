package daemon

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/rgb-24bit/taskdeck/internal/config"
	"github.com/rgb-24bit/taskdeck/internal/server"
	"github.com/rgb-24bit/taskdeck/internal/store"
)

const maxLogSize = 10 * 1024 * 1024

func Run(cfg *config.Config) error {
	if err := config.EnsureDir(); err != nil {
		return fmt.Errorf("ensure config dir: %w", err)
	}

	// Daemonize: fork to background
	if os.Getenv("TASKDECK_DAEMON") != "1" {
		return daemonize()
	}

	// We're in the child process
	if err := writePID(cfg.PidPath); err != nil {
		return fmt.Errorf("write pid: %w", err)
	}
	defer os.Remove(cfg.PidPath)

	logStop := setupLog(cfg.LogPath)

	st, err := store.New(cfg.DBPath)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer st.Close()

	handler := server.New(st)

	// Stop channels for background goroutines
	done := make(chan struct{})

	go timeoutChecker(st, 30*time.Second, done)

	addr := fmt.Sprintf(":%d", cfg.Port)
	httpSrv := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	// Signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("shutting down...")
		close(done)
		close(logStop)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := httpSrv.Shutdown(ctx); err != nil {
			log.Printf("shutdown error: %v", err)
		}
		st.Close()
		os.Remove(cfg.PidPath)
		os.Exit(0)
	}()

	log.Printf("taskdeck serving on http://localhost%s", addr)
	if err := httpSrv.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}
	return nil
}

func daemonize() error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable: %w", err)
	}

	cmd := exec.Command(exe, os.Args[1:]...)
	cmd.Env = append(os.Environ(), "TASKDECK_DAEMON=1")
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}

	return cmd.Start()
}

func writePID(path string) error {
	return os.WriteFile(path, []byte(strconv.Itoa(os.Getpid())), 0644)
}

func ReadPID(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(string(data))
}

func IsRunning(path string) bool {
	pid, err := ReadPID(path)
	if err != nil {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}

func setupLog(path string) chan struct{} {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("cannot open log: %v", err)
		return make(chan struct{})
	}
	log.SetOutput(io.MultiWriter(f, os.Stderr))
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	stop := make(chan struct{})
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				stat, err := os.Stat(path)
				if err != nil {
					continue
				}
				if stat.Size() > maxLogSize {
					os.Rename(path, path+".1")
					newF, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
					if err == nil {
						log.SetOutput(io.MultiWriter(newF, os.Stderr))
					}
				}
			case <-stop:
				return
			}
		}
	}()
	return stop
}

func timeoutChecker(st *store.Store, interval time.Duration, done <-chan struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			tasks, err := st.GetExpiredWaiting()
			if err != nil {
				log.Printf("timeout check error: %v", err)
				continue
			}
			for _, t := range tasks {
				if _, err := st.Activate(t.ID); err != nil {
					log.Printf("auto-activate %d error: %v", t.ID, err)
				} else {
					log.Printf("auto-activated task %d: %s", t.ID, t.Title)
				}
			}
		case <-done:
			return
		}
	}
}
