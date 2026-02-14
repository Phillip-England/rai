package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
)

var colors = []string{
	"\033[36m", // cyan
	"\033[35m", // magenta
	"\033[33m", // yellow
	"\033[32m", // green
	"\033[34m", // blue
}

const reset = "\033[0m"

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "usage: rai 'cmd1 args...' 'cmd2 args...' ...\n")
		os.Exit(1)
	}

	var (
		mu       sync.Mutex
		cmds     []*exec.Cmd
		wg       sync.WaitGroup
		exitCode int
		done     = make(chan struct{})
		once     sync.Once
	)

	killAll := func(skip int) {
		mu.Lock()
		defer mu.Unlock()
		for i, c := range cmds {
			if i == skip || c.Process == nil {
				continue
			}
			// Kill the entire process group
			syscall.Kill(-c.Process.Pid, syscall.SIGTERM)
		}
		go func() {
			<-done // wait for all goroutines to finish naturally, or timeout
		}()
	}

	// Forward signals to all children
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig, ok := <-sigCh
		if !ok {
			return
		}
		mu.Lock()
		for _, c := range cmds {
			if c.Process != nil {
				syscall.Kill(-c.Process.Pid, sig.(syscall.Signal))
			}
		}
		mu.Unlock()
		os.Exit(1)
	}()

	for i, arg := range args {
		label := strings.Fields(arg)[0]
		color := colors[i%len(colors)]
		prefix := fmt.Sprintf("%s[%s]%s ", color, label, reset)

		cmd := exec.Command("sh", "-c", arg)
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

		mu.Lock()
		cmds = append(cmds, cmd)
		mu.Unlock()

		if err := cmd.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "error starting %q: %v\n", arg, err)
			os.Exit(1)
		}

		idx := i

		// Stream stdout
		wg.Add(1)
		go func() {
			defer wg.Done()
			scanner := bufio.NewScanner(stdout)
			for scanner.Scan() {
				fmt.Printf("%s%s\n", prefix, scanner.Text())
			}
		}()

		// Stream stderr
		wg.Add(1)
		go func() {
			defer wg.Done()
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				fmt.Fprintf(os.Stderr, "%s%s\n", prefix, scanner.Text())
			}
		}()

		// Wait for command completion
		wg.Add(1)
		go func(c *exec.Cmd, cmdIdx int) {
			defer wg.Done()
			if err := c.Wait(); err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					once.Do(func() {
						exitCode = exitErr.ExitCode()
						killAll(cmdIdx)
					})
				}
			}
		}(cmd, idx)
	}

	wg.Wait()
	close(done)
	signal.Stop(sigCh)
	close(sigCh)
	os.Exit(exitCode)
}
