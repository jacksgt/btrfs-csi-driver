package driver

func execWithLog(args ...string) (*exec.Cmd) {
	klog.V(6).Info("Executing command: ", args...)
	return exec.Command(args...)
}
