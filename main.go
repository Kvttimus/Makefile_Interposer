package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
)

//1. (DONE) fork build process
//2. (IN PROGRESS) trace syscall - need to trace all forks/clones
//3. () parse/interpret args from syscall
//4. () modify command (replace args in tracee memory by modifying registers)
//5. () resume the process

// TODO: Follow forks and clones
// Find actual arguments (execve args)

// GOAL - trace all exec related syscalls
// LOOK INTO GO

func procName(pid int) string {
	b, err := os.ReadFile(fmt.Sprintf("/proc/%d/comm", pid))
	if (err != nil) {  // process gone/unreachable :o
		return "?"
	}
	return strings.TrimSpace(string(b))  // bytes-->string
}

func main() {
	fmt.Printf("Number argc args: %d\n", len(os.Args))
	
	if len(os.Args) < 2 {
		fmt.Println("Need to pass in command to trace")
		return
	}

	runtime.LockOSThread() // pin go tracer to 1 program thread

	// Step 1. Fork the process
	cmd := exec.Command(os.Args[1])
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Ptrace: true, // equiv of ptrace(PTRACE_TRACEME) in C + stops at exec
	}

	// fork + exec child
	err := cmd.Start()
	if err != nil {
		fmt.Printf("fork failed: %v", err)
		return
	}

	// Step 2. Trace syscall
	childPid := cmd.Process.Pid
	var status syscall.WaitStatus  // wait res
	var regs syscall.PtraceRegs  // reg snapshot

	syscall.Wait4(childPid, &status, 0, nil)  // catch child init. stop

	for {
		err := syscall.PtraceSyscall(childPid, 0)
		if (err != nil) { // resume until next syscall boundary
			fmt.Println("SYSCALL failed")
			break
		}

		_, err = syscall.Wait4(childPid, &status, 0, nil)
		if (err != nil) {  // stop at syscall boundary
			fmt.Println("waitpid failed")
			break
		}

		if (status.Exited() || status.Signaled()) {  // child exits normally/sigkill
			fmt.Println("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA")
			break
		}

		if (syscall.PtraceGetRegs(childPid, &regs) == nil) {
			fmt.Printf("[%s pid %d] hit syscall id: %d\n", procName(childPid), childPid, regs.Orig_rax);
		}
	}
}