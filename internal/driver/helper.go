package driver

import (
	"os/exec"

	"k8s.io/klog/v2"
)

func execWithLog(args ...string) (*exec.Cmd) {
	klog.V(6).Info("Executing command: ", args)
	if len(args) == 0 {
		return nil
	}
	if len(args) == 1 {
		return exec.Command(args[0])
	}
	return exec.Command(args[0], args[1:]...)
}
