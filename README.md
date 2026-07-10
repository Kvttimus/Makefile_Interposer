# Makefile_Interposer

NOTE TO SELF
- to test system call tracing, change `cmd := exec.Command(os.Args[1])` to `cmd := exec.Command("./testExecve")`
- then rebuild main.go and then run it as normal