package debugutils

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"

	errors "github.com/rotisserie/eris"
)


type NamespacedDumpCommandGenerator func(namespace string) []string

func KubeDumpOnFail(out io.Writer, namespaces []string, additionalCommands NamespacedDumpCommandGenerator) func() {
	return func() {
		PrintDockerState()
		PrintProcessState()
		dump, err := KubeDump(namespaces, additionalCommands)
		if err != nil {
			fmt.Fprintf(out, "getting kube dump failed: %v", err)
		}
		fmt.Fprintf(out, dump)
	}
}

// dump all data from the kube cluster
func KubeDump(namespaces []string, additionalCommands NamespacedDumpCommandGenerator) (string, error) {
	b := &bytes.Buffer{}
	b.WriteString("** Begin Kubernetes Dump ** \n")
	for _, ns := range namespaces {
		for _, command := range additionalCommands(ns) {
			cmd := exec.Command("bash", "-c", command)
			cmd.Stdout = b
			cmd.Stderr = b
			if err := cmd.Run(); err != nil {
				return "", errors.Errorf("command %v failed: %v", command, b.String())
			}
		}
	}
	b.WriteString("** End Kubernetes Dump ** \n")
	return b.String(), nil
}

func PrintDockerState() {
	dockerCmd := exec.Command("docker", "ps")

	dockerState := &bytes.Buffer{}

	dockerCmd.Stdout = dockerState
	dockerCmd.Stderr = dockerState
	err := dockerCmd.Run()
	if err != nil {
		fmt.Println("*** Unable to get docker state ***")
		return
	}
	fmt.Println("*** Docker state ***")
	fmt.Println(dockerState.String())
	fmt.Println("*** End Docker state ***")
}

func PrintProcessState() {
	psCmd := exec.Command("ps", "-auxf")

	psState := &bytes.Buffer{}

	psCmd.Stdout = psState
	psCmd.Stderr = psState
	err := psCmd.Run()
	if err != nil {
		fmt.Println("*** Unable to get process state ***")
		return
	}
	fmt.Println("*** Process state ***")
	fmt.Println(psState.String())
	fmt.Println("*** End Process state ***")
}