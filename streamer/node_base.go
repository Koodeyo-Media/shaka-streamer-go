package streamer

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

// ProcessStatus describes the status of a process.
type ProcessStatus int

const (
	// Finished means the node has completed its task and shut down.
	Finished ProcessStatus = iota

	// Running means the node is still running.
	Running

	// Errored means the node has failed.
	Errored
)

func formatEnv(env map[string]string) []string {
	formatted := make([]string, 0, len(env))
	for k, v := range env {
		formatted = append(formatted, fmt.Sprintf("%v=%v", k, v))
	}
	return formatted
}

// NodeBase is a base class for nodes that run a single subprocess.
type NodeBase struct {
	Process *exec.Cmd
}

type BaseParams struct {
	args     []string
	env      map[string]string
	mergeEnv bool
	stdout   io.Writer
	stderr   io.Writer
}

// Start should be overridden by the subclass to construct a command line, call
// createProcess, and assign the result to process.
func (nb *NodeBase) Start() {
	panic("NodeBase.Start() should be overridden")
}

/*
A central point to create subprocesses, so that we can debug thecommand-line arguments.

Args:

	args: An array of strings if shell is False, or a single string is shell is True; the command line of the subprocess.
	env: A dictionary of environment variables to pass to the subprocess.
	merge_env: If true, merge env with the parent process environment.
	shell: If true, args must be a single string, which will be executed as a shell command.

Returns: The Popen object of the subprocess.
*/
func (nb *NodeBase) CreateProcess(params BaseParams) *exec.Cmd {
	cmd := exec.Command(params.args[0], params.args[1:]...)

	if params.mergeEnv {
		cmd.Env = append(os.Environ(), formatEnv(params.env)...)
	} else {
		cmd.Env = formatEnv(params.env)
	}

	// Print arguments formatted as output from bash -x would be.
	// This makes it easy to see the arguments and easy to copy/paste them for
	// debugging in a shell.
	fmt.Printf("+ %s\n", strings.Join(params.args, " "))

	cmd.Stdin = nil
	cmd.Stdout = params.stdout
	cmd.Stderr = params.stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		panic(err)
	}

	// if err := cmd.Run(); err != nil {
	// 	panic(err)
	// }

	// if err := cmd.Wait(); err != nil {
	// 	panic(err)
	// }

	return cmd
}

// Returns the current ProcessStatus of the node.
func (nb *NodeBase) CheckStatus() ProcessStatus {
	if nb.Process == nil {
		panic("Must have a process to check")
	}

	if nb.Process.ProcessState != nil && nb.Process.ProcessState.Exited() {
		if nb.Process.ProcessState.Success() {
			return Finished
		} else {
			return Errored
		}
	}

	return Running
}

// Stop the subprocess if it's still running.
func (nb *NodeBase) Stop() {
	if nb.Process != nil {
		// Slightly more polite than kill.  Try this first.
		pgid, err := syscall.Getpgid(nb.Process.Process.Pid)
		if err == nil {
			syscall.Kill(-pgid, syscall.SIGTERM)
		}

		if nb.CheckStatus() == Running {
			// If it's not dead yet, wait 1 second.
			time.Sleep(time.Second)
		}

		if nb.CheckStatus() == Running {
			// If it's still not dead, use kill.
			pgid, err := syscall.Getpgid(nb.Process.Process.Pid)
			if err == nil {
				syscall.Kill(-pgid, syscall.SIGKILL)
			}

			// Wait for the process to die and read its exit code. There is no way
			// to ignore a kill signal, so this will happen quickly. If we don't do
			// this, it can create a zombie process.
			nb.Process.Wait()
		}
	}
}
