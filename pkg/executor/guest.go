// Copyright 2019 Hewlett Packard Enterprise Development LP

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package executor

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/bluek8s/kubedirector/pkg/observer"
	"github.com/bluek8s/kubedirector/pkg/shared"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/client-go/util/exec"
)

// IsFileExists probes whether the given pod's filesystem contains something
// at the indicated filepath. The returned boolean will be true if the file
// was found. If false, the returned error will be nil if the file is known to
// be missing, or non-nil if the probe failed to execute.
func IsFileExists(
	args *ArgumentSet,
	filePath string,
	isDirectory bool,
) (bool, error) {

	testExpr := "-f"
	if isDirectory {
		testExpr = "-d"
	}
	command := []string{"test", testExpr, filePath}
	// We only need the exit status, but we have to supply at least one
	// stream to avoid an error.
	var stdOut bytes.Buffer
	ioStreams := &Streams{Out: &stdOut}
	execErr := ExecCommand(
		args,
		command,
		ioStreams,
	)
	if execErr != nil {
		// Determine which type of error occured
		coe, iscoe := execErr.(exec.CodeExitError)
		if iscoe {
			// If the command failed with a CodeExitError error and an exit
			// code of 1, this means that the file existence check completed
			// successfully, but the file does not exist.
			if coe.ExitStatus() == 1 {
				return false, nil
			}
		}
		// Some error, other than file does not exist, occured.
		return false, execErr
	}
	// The file exists.
	return true, nil
}

// CreateDir creates a directory (and any parent directories) as necessary in
// the filesystem of the given pod. If the setPerms option is true, the
// directory will have its permissions set to 700.
func CreateDir(
	args *ArgumentSet,
	dirName string,
	setPerms bool,
) error {

	command := []string{"mkdir", "-p", dirName}
	// We only need the exit status, but we have to supply at least one
	// stream to avoid an error.
	var stdErr bytes.Buffer
	ioStreams := &Streams{ErrOut: &stdErr}
	err := ExecCommand(
		args,
		command,
		ioStreams,
	)
	if err != nil {
		err = fmt.Errorf("mkdir failed: %s\n%s",
			stdErr.String(),
			err.Error(),
		)
		return err
	}
	if !setPerms {
		return nil
	}
	command = []string{"chmod", "700", dirName}
	err = ExecCommand(
		args,
		command,
		ioStreams,
	)
	if err != nil {
		err = fmt.Errorf("directory chmod failed: %s\n%s",
			stdErr.String(),
			err.Error(),
		)
	}
	return err
}

// RemoveDir removes a directory.
func RemoveDir(
	args *ArgumentSet,
	dirName string,
	ignoreNotEmpty bool,
) error {

	var command []string
	if ignoreNotEmpty {
		command = []string{"rmdir", "--ignore-fail-on-non-empty", dirName}
	} else {
		command = []string{"rmdir", dirName}
	}

	// We only need the exit status, but we have to supply at least one
	// stream to avoid an error.
	var stdErr bytes.Buffer
	ioStreams := &Streams{ErrOut: &stdErr}
	err := ExecCommand(
		args,
		command,
		ioStreams,
	)
	if err != nil {
		errStr := stdErr.String()
		if strings.Contains(errStr, "No such file or directory") {
			err = nil
		} else {
			err = fmt.Errorf("rmdir failed: %s", errStr)
		}
	}
	return err
}

// CreateFile takes the stream from the given reader, and writes it to the
// indicated filepath in the filesystem of the given pod. Parent directories
// will be created as needed. If the setDirPerms option is true, the directory
// containing the file will have its permissions set to 700.
func CreateFile(
	args *ArgumentSet,
	filePath string,
	reader io.Reader,
	setDirPerms bool,
) error {

	createDirErr := CreateDir(
		args,
		filepath.Dir(filePath),
		setDirPerms,
	)
	if createDirErr != nil {
		return createDirErr
	}

	command := []string{"tee", filePath}
	ioStreams := &Streams{
		In: reader,
	}
	shared.LogInfof(
		args.Logger,
		args.Cluster,
		shared.EventReasonNoEvent,
		"creating file{%s} in pod{%s}",
		filePath,
		args.PodName,
	)
	return ExecCommand(
		args,
		command,
		ioStreams,
	)
}

// ReadFile takes the stream from the given writer, and writes to it the
// contents of the indicated filepath in the filesystem of the given pod.
// The returned boolean and error are interpreted in the same way as fo
// IsFileExists.
func ReadFile(
	args *ArgumentSet,
	filePath string,
	writer io.Writer,
) (bool, error) {

	command := []string{"cat", filePath}
	ioStreams := &Streams{
		Out: writer,
	}
	shared.LogInfof(
		args.Logger,
		args.Cluster,
		shared.EventReasonNoEvent,
		"reading file{%s} in pod{%s}",
		filePath,
		args.PodName,
	)
	execErr := ExecCommand(
		args,
		command,
		ioStreams,
	)
	if execErr != nil {
		coe, iscoe := execErr.(exec.CodeExitError)
		if iscoe {
			if coe.ExitStatus() == 1 {
				return false, nil
			}
		}
		return false, execErr
	}
	return true, nil
}

// RunScript takes the stream from the given reader, and executes it as a
// shell script in the given pod.
func RunScript(
	args *ArgumentSet,
	description string,
	reader io.Reader,
) error {

	command := []string{execShell}
	ioStreams := &Streams{
		In: reader,
	}
	shared.LogInfof(
		args.Logger,
		args.Cluster,
		shared.EventReasonNoEvent,
		"running %s in pod{%s}",
		description,
		args.PodName,
	)
	return ExecCommand(
		args,
		command,
		ioStreams,
	)
}

// ExecCommand is a utility function for executing a command in a pod. It
// uses the given ioStreams to provide the command inputs and accept the
// command outputs.
func ExecCommand(
	args *ArgumentSet,
	command []string,
	ioStreams *Streams,
) error {

	pod, podErr := observer.GetPod(args.NameSpace, args.PodName)
	if podErr != nil {
		shared.LogErrorf(
			args.Logger,
			podErr,
			args.Cluster,
			shared.EventReasonNoEvent,
			"could not find pod{%s}",
			args.PodName,
		)
		return fmt.Errorf(
			"pod{%v} does not exist",
			args.PodName,
		)
	}

	foundContainer := false
	for _, containerStatus := range pod.Status.ContainerStatuses {
		if containerStatus.Name == args.ContainerName {
			foundContainer = true
			if containerStatus.ContainerID != args.ContainerID {
				return errors.New("container ID changed during configuration")
			}
			break
		}
	}
	if !foundContainer {
		return fmt.Errorf(
			"container{%s} does not exist in pod{%v}",
			args.ContainerName,
			args.PodName,
		)
	}

	if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
		return fmt.Errorf(
			"cannot connect to pod{%v} in phase %v",
			args.PodName,
			pod.Status.Phase,
		)
	}

	request := shared.ClientSet().CoreV1().RESTClient().Post().
		Resource("pods").
		Name(args.PodName).
		Namespace(args.NameSpace).
		SubResource("exec").
		Param("container", args.ContainerName)
	request.VersionedParams(&corev1.PodExecOptions{
		Container: args.ContainerName,
		Command:   command,
		Stdin:     ioStreams.In != nil,
		Stdout:    ioStreams.Out != nil,
		Stderr:    ioStreams.ErrOut != nil,
	}, scheme.ParameterCodec)

	exec, initErr := remotecommand.NewSPDYExecutor(
		shared.Config(),
		"POST",
		request.URL(),
	)
	if initErr != nil {
		shared.LogError(
			args.Logger,
			initErr,
			args.Cluster,
			shared.EventReasonNoEvent,
			"failed to init the executor",
		)
		return errors.New("failed to initialize command executor")
	}
	execErr := exec.Stream(remotecommand.StreamOptions{
		Tty:    false,
		Stdin:  ioStreams.In,
		Stdout: ioStreams.Out,
		Stderr: ioStreams.ErrOut,
	})

	return execErr
}
