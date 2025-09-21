package mount

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
	"k8s.io/mount-utils"

	"github.com/3fs-csi/3fs-csi/pkg/config"
)

type Manager interface {
	EnsureNodeMount(ctx context.Context) error
	BindMountIfNeeded(source, target string) error
	Unmount(target string) error
	IsMounted(target string) (bool, error)
}

type manager struct {
	cfg     *config.Config
	mounter mount.Interface
}

func New(cfg *config.Config) (Manager, error) {
	return &manager{
		cfg:     cfg,
		mounter: mount.New(""),
	}, nil
}

func (m *manager) EnsureNodeMount(ctx context.Context) error {
	mountPoint := m.cfg.GlobalMountPoint()
	if err := os.MkdirAll(mountPoint, 0755); err != nil {
		return fmt.Errorf("mkdir global mount: %w", err)
	}
	if err := os.MkdirAll(m.cfg.ConfigDir, 0755); err != nil {
		return fmt.Errorf("mkdir config dir: %w", err)
	}

	locked, unlock, err := m.acquireLock()
	if err != nil {
		return err
	}
	defer func() {
		if unlock != nil {
			_ = unlock()
		}
	}()
	if !locked {
		return m.waitForMount(ctx, mountPoint, 60*time.Second)
	}

	// If already mounted, nothing to do
	mounted, err := m.isMountPoint(mountPoint)
	if err != nil {
		return err
	}
	if mounted {
		return nil
	}

	if err := m.renderLauncher(); err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, m.cfg.HF3FSBinaryPath, "-cfg", m.cfg.LauncherPath())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	logrus.WithFields(logrus.Fields{
		"bin":   m.cfg.HF3FSBinaryPath,
		"cfg":   m.cfg.LauncherPath(),
		"mount": mountPoint,
	}).Info("starting hf3fs_fuse_main")

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start hf3fs_fuse_main: %w", err)
	}

	if err := m.waitForMount(ctx, mountPoint, 60*time.Second); err != nil {
		return err
	}
	return nil
}

func (m *manager) BindMountIfNeeded(source, target string) error {
	if err := os.MkdirAll(target, 0755); err != nil {
		return fmt.Errorf("mkdir target: %w", err)
	}
	if err := os.MkdirAll(source, 0775); err != nil {
		return fmt.Errorf("mkdir source: %w", err)
	}
	ismnt, err := m.isMountPoint(target)
	if err != nil {
		return err
	}
	if ismnt {
		return nil
	}
	opts := []string{"rbind"}
	if err := m.mounter.Mount(source, target, "", opts); err != nil {
		return fmt.Errorf("bind mount %s -> %s: %w", source, target, err)
	}
	_ = makeRShared(target)
	return nil
}

func (m *manager) Unmount(target string) error {
	ismnt, err := m.isMountPoint(target)
	if err != nil {
		return err
	}
	if !ismnt {
		return nil
	}
	if err := m.mounter.Unmount(target); err != nil {
		return fmt.Errorf("unmount %s: %w", target, err)
	}
	return nil
}

func (m *manager) IsMounted(target string) (bool, error) { return m.isMountPoint(target) }

func (m *manager) isMountPoint(path string) (bool, error) {
	if _, err := os.Stat(path); err != nil && os.IsNotExist(err) {
		return false, nil
	}
	notMnt, err := mount.New(""
	).IsLikelyNotMountPoint(path)
	if err != nil {
		return false, err
	}
	return !notMnt, nil
}

func (m *manager) renderLauncher() error {
	content := renderLauncherTOML(m.cfg)
	path := m.cfg.LauncherPath()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("write launcher toml: %w", err)
	}
	return nil
}

func (m *manager) waitForMount(ctx context.Context, mountPoint string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		mounted, err := m.isMountPoint(mountPoint)
		if err == nil && mounted {
			return nil
		}
		if time.Now().After(deadline) {
			if err != nil {
				return fmt.Errorf("timed out waiting for fuse mount: %w", err)
			}
			return errors.New("timed out waiting for fuse mount")
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func (m *manager) acquireLock() (bool, func() error, error) {
	lockFile := filepath.Join(m.cfg.ConfigDir, fmt.Sprintf("%s.mount.lock", m.cfg.ClusterID))
	f, err := os.OpenFile(lockFile, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return false, nil, fmt.Errorf("open lock file: %w", err)
	}
	if err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		// can't acquire
		_ = f.Close()
		if errors.Is(err, unix.EWOULDBLOCK) {
			return false, nil, nil
		}
		return false, nil, fmt.Errorf("flock: %w", err)
	}
	return true, func() error {
		defer f.Close()
		return unix.Flock(int(f.Fd()), unix.LOCK_UN)
	}, nil
}

func makeRShared(target string) error {
	shared, err := isSharedMount(target)
	if err != nil {
		return err
	}
	if shared {
		return nil
	}
	cmd := exec.Command("mount", "--make-rshared", target)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func isSharedMount(target string) (bool, error) {
	f, err := os.Open("/proc/self/mountinfo")
	if err != nil {
		return false, err
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := s.Text()
		// fields: https://man7.org/linux/man-pages/man5/proc.5.html
		fields := strings.Split(line, " - ")
		if len(fields) < 2 {
			continue
		}
		pfx := fields[0]
		parts := strings.Split(pfx, " ")
		if len(parts) < 5 {
			continue
		}
		mountPoint := parts[4]
		if mountPoint != target {
			continue
		}
		for _, f := range parts[6:] {
			if strings.HasPrefix(f, "shared:") {
				return true, nil
			}
		}
	}
	return false, s.Err()
}


