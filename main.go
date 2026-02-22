package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

const (
	ubuntuURL        = "https://cdimage.ubuntu.com/ubuntu-base/releases/22.04/release/ubuntu-base-22.04-base-amd64.tar.gz"
	rootfsDir string = "rootfs"
)

// main is the entry point of the program.
// It checks the command-line arguments and decides whether to call run or child.
func main() {
	// If there are fewer than 2 arguments (only the program name),
	// print a message and exit.
	if len(os.Args) < 2 {
		fmt.Println("args >1 ")
		return
	}

	// Decide what to do based on the first argument (os.Args[1]).
	switch os.Args[1] {
	case "init":
		fmt.Println("Initializing rootfs with Ubuntu...")
		must(Rootfs(rootfsDir, ubuntuURL))
		return
	case "run":
		// If the first argument is "run", call the run() function (parent process).
		// The second argument is the rootfs path (e.g., "rootfs/alpine").
		run()
	case "child":
		// child expects: child <rootfs_path> <cmd> [args...]
		child()
	default:
		// If the first argument is anything else, panic with an error message.
		panic("Unknown command. Use: alpine | ubuntu | run <rootfs_path> <cmd>")
	}
}

// run is the parent process function.
// It spawns a new child process that runs the same binary with the "child" argument.
func run() {
	fmt.Printf("RUN PROCESS ID: PID=%d\n", os.Getpid())

	// Create a command that runs the current binary again:
	// /proc/self/exe refers to the current executable (this program itself).
	// "child" becomes the first argument in the child process.
	cmd := exec.Command("/proc/self/exe", append([]string{"child"}, os.Args[2:]...)...)
	// Connect the child process's stdin/stdout/stderr to the current terminal.
	// This makes the child process interactive:
	// - Whatever you type in the terminal goes to the child.
	// - Whatever the child prints goes directly to the terminal.
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// namespace
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWPID |
			syscall.CLONE_NEWUTS |
			syscall.CLONE_NEWNS |
			syscall.CLONE_NEWNET |
			syscall.CLONE_NEWIPC |
			syscall.CLONE_NEWUSER |
			syscall.CLONE_NEWCGROUP,

		UidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID:      os.Getuid(),
				Size:        1,
			},
		},
		GidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID:      os.Getgid(),
				Size:        1,
			},
		},
		GidMappingsEnableSetgroups: false,
	}

	// cmd.Run() starts the child process and waits for it to finish.
	// If there is an error, must(err) will panic and stop the program.
	must(cmd.Run())
}

// child is the child process function.
// It sets up the container environment and runs the given command.
func child() {
	fmt.Printf("CHILD PROCESS ID: PID=%d\n", os.Getpid())

	must(syscall.Sethostname([]byte("cfm-container")))
	must(syscall.Chroot("./rootfs"))
	must(os.Chdir("/"))
	must(syscall.Mount("proc", "proc", "proc", 0, ""))
	cg()

	// Fix: command is at index 2, not 3
	if len(os.Args) < 3 {
		fmt.Println("child: no command given, running /bin/sh")
		os.Args = append(os.Args, "/bin/sh")
	}

	command := os.Args[2] // was os.Args[3]
	args := os.Args[3:]   // was os.Args[4:]

	cmd := exec.Command(command, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	must(cmd.Run())
	must(syscall.Unmount("proc", 0))
}

// cg sets up a cgroup to limit CPU and memory for the container.
func cg() {
	cgroupPath := "/sys/fs/cgroup/minicontainer"
	must(os.MkdirAll(cgroupPath, 0755))

	// 100 MB memory limit
	must(os.WriteFile(cgroupPath+"/memory.max", []byte("100000000"), 0700))

	// 50% of one CPU core
	must(os.WriteFile(cgroupPath+"/cpu.max", []byte("50000 100000"), 0700))

	// Add current process to the cgroup
	must(os.WriteFile(cgroupPath+"/cgroup.procs", []byte(fmt.Sprintf("%d", os.Getpid())), 0700))
}

// Rootfs downloads and extracts a minirootfs tarball into rootfs.
func Rootfs(rootfs string, url string) error {
	if err := os.MkdirAll(rootfs, 0755); err != nil {
		return fmt.Errorf("failed to create rootfs dir: %w", err)
	}

	archivePath := filepath.Join(rootfs, "minirootfs.tar.gz")

	if _, err := os.Stat(archivePath); os.IsNotExist(err) {
		fmt.Printf("2. Downloading minirootfs from %s...\n", url)
		response, err := http.Get(url)
		if err != nil {
			return fmt.Errorf("failed to download minirootfs: %w", err)
		}
		defer response.Body.Close()

		if response.StatusCode != http.StatusOK {
			return fmt.Errorf("bad status: %s", response.Status)
		}

		out, err := os.Create(archivePath)
		if err != nil {
			return fmt.Errorf("failed to create archive file: %w", err)
		}
		defer out.Close()

		_, err = io.Copy(out, response.Body)
		if err != nil {
			return fmt.Errorf("failed to write archive: %w", err)
		}

		fmt.Printf("Downloaded: %s\n", archivePath)
	} else {
		fmt.Printf("2. Archive already exists: %s, skipping download.\n", archivePath)
	}

	// 2. Extract tar.gz
	fmt.Printf("3. Extracting rootfs to %s...\n", rootfs)

	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar error: %w", err)
		}

		// Clean path to prevent ../ attacks
		cleanPath := filepath.Clean(header.Name)
		if strings.Contains(cleanPath, "..") {
			return fmt.Errorf("invalid path: %s", cleanPath)
		}

		// Target path inside rootfs
		target := filepath.Join(rootfs, cleanPath)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return fmt.Errorf("failed to create dir %s: %w", target, err)
			}
		case tar.TypeReg:
			outFile, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, header.FileInfo().Mode()) // ← Mode() qo'shing
			if err != nil {
				return fmt.Errorf("failed to create file %s: %w", target, err)
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return fmt.Errorf("failed to extract file %s: %w", target, err)
			}
			outFile.Close()
		case tar.TypeSymlink: // ← QO'SHING
			os.Remove(target)
			if err := os.Symlink(header.Linkname, target); err != nil {
				return fmt.Errorf("failed to create symlink %s -> %s: %w", target, header.Linkname, err)
			}
		default:
			continue
		}
	}

	// 3. Clean up archive
	fmt.Printf("4. Cleaning up archive...\n")
	if err := os.Remove(archivePath); err != nil {
		return fmt.Errorf("failed to remove archive: %w", err)
	}

	fmt.Printf("5. Done! Rootfs is ready in ./%s\n", rootfs)
	return nil
}

// must is a helper function to check errors.
// If err is not nil, it panics and stops the program.
func must(err error) {
	if err != nil {
		panic(err)
	}
}
