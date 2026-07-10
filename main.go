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
//2. (DONE) trace syscall - need to trace all forks/clones
//3. (IN PROGRESS) parse/interpret args from syscall
//4. () modify command (replace args in tracee memory by modifying registers)
//5. () resume the process

// Find actual arguments (execve args)
// execve signature: int execve(const char *pathname, char *const argv[], char *const envp[]);
/*
- Sys Call ID/Ret Val - %rax
- Arg1 - %rdi
- Arg2 - %rsi
- Arg3 - %rdx
- Arg4 - %r10
- Arg5 - %r8
- Arg6 - %r9

rdi → pathname
rsi → argv (arg vector array)
rdx → envp (env var array)

STEOS
- fetch registers to locate * arrays in tracee addr sapce
- read target strings using PTRACE_PEEKDATA or /proc/pid/mem access
- 
*/


// GOAL - trace all exec related syscalls

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

	runtime.LockOSThread() // pin go tracer to 1 program thread (req.)

	// STEP 1. Fork the process  -1-1-1-1-1-1-1-1-1-1-1-1-1-1-1-1-1-1-1-1 ONE ONE ONE ONE ONE ONE
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

	childPid := cmd.Process.Pid
	var status syscall.WaitStatus  // wait res
	var regs syscall.PtraceRegs  // reg snapshot
	
	// STEP 2. Trace syscall -2-2-2-2-2-2-2-2-2-2-2-2-2-2-2-2-2-2-2-2 TWO TWO TWO TWO TWO TWO
	syscall.Wait4(childPid, &status, 0, nil)  // catch child init. stop

	options := syscall.PTRACE_O_TRACEFORK |  // auto attach + stop any new child
		syscall.PTRACE_O_TRACEVFORK |
		syscall.PTRACE_O_TRACECLONE |
		// changes the 7th bit to differentiate syscall start/stop vs other sigtraps
		syscall.PTRACE_O_TRACESYSGOOD  
	syscall.PtraceSetOptions(childPid, options)

	syscall.PtraceSyscall(childPid, 0)

	for {
		// wait for event from any child (-1)
		pid, err := syscall.Wait4(-1, &status, 0, nil)
		if (err != nil) {  // traced all processes inc. fork/clone
			fmt.Println("No traced processes left")
			break
		}

		if (status.Exited() || status.Signaled()) {  // cur pid died -> cont. to next
			fmt.Println("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA")
			continue
		}

		sig := status.StopSignal()

		if (sig == syscall.SIGTRAP | 0x80) {  // syscall enter/exit boundary
			if (syscall.PtraceGetRegs(pid, &regs) == nil) {
				fmt.Printf("[%s pid %d] hit syscall id: %d\n", procName(childPid), childPid, regs.Orig_rax)
			}
			syscall.PtraceSyscall(pid, 0)
		} else if (status.TrapCause() != -1) {  // fork event stops on parent (child created) -> resume it
			syscall.PtraceSyscall(pid, 0)
		} else if (sig == syscall.SIGTRAP || sig == syscall.SIGSTOP) {  // other sigtraps/sigstops
			syscall.PtraceSyscall(pid, 0)
		} else {  // signal aimed @ tracee -> resume + reinject
			syscall.PtraceSyscall(pid, int(sig))
		}
	}
}