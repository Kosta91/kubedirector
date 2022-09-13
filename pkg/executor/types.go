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
	"io"

	kdv1 "github.com/bluek8s/kubedirector/pkg/apis/kubedirector/v1beta1"
	"github.com/bluek8s/kubedirector/pkg/shared"
	"github.com/go-logr/logr"
)

const (
	// ClusterAppLabel is a label placed on every created statefulset, pod,
	// and service, with a value of the KubeDirectorApp CR name.
	ClusterAppLabel = shared.KdDomainBase + "/kdapp"
	// ClusterAppCatalogLabel is a label placed on every created statefulset,
	// pod, and service, with a value "local" or "system" appropriately.
	ClusterAppCatalogLabel = shared.KdDomainBase + "/appCatalog"
	// ClusterRoleLabel is a label placed on every created pod, and
	// (non-headless) service, with a value of the relevant role ID.
	ClusterRoleLabel = shared.KdDomainBase + "/role"
	// HeadlessServiceLabel is a label placed on the statefulset and pods.
	// Used in a selector on the headless service.
	HeadlessServiceLabel = shared.KdDomainBase + "/headless"

	// ClusterAppAnnotation is an annotation placed on every created
	// statefulset, pod, and service, with a value of the KubeDirectorApp's
	// spec.label.name.
	ClusterAppAnnotation = shared.KdDomainBase + "/kdapp-prettyName"

	statefulSetPodLabel = "statefulset.kubernetes.io/pod-name"
	// AppContainerName is the name of KubeDirector app containers.
	AppContainerName = "app"
	// PvcNamePrefix (along with a hyphen) is prepended to the name of each
	// member PVC name that is auto-created for a statefulset.
	PvcNamePrefix         = "p"
	svcNamePrefix         = "s-"
	statefulSetNamePrefix = "kdss-"
	headlessSvcNamePrefix = "kdhs-"
	initContainerName     = "init"
	execShell             = "bash"
	configMetaFile        = "/etc/guestconfig/configmeta.json"
	cgroupFSVolume        = "/sys/fs/cgroup"
	systemdFSVolume       = "/sys/fs/cgroup/systemd"
	tmpFSVolSize          = "20Gi"
	kubedirectorInit      = "/etc/kubedirector.init"
	// The file that contains full logs of copying persistent dirs
	kubedirectorInitLogs = "/etc/kubedirector-init.log"
	// The file that contains just a progress bar of copying persisten dirs
	// The file is updated dynamically
	kubedirectorInitProgressBar = "/etc/kubedirector-init-progress-bar.log"

	// nvidiaGpuResourcePrefix is the name of a GPU resource, schedulable for a container -
	// specifically, a GPU by the vendor, NVIDIA
	nvidiaGpuResourcePrefix = "nvidia.com/"
	// nvidiaGpuVisWorkaroundEnvVarName is the name of an environment variable, which is to be
	// injected in a scheduled container), as an NVIDIA-suggested work-around that
	// avoids an NVIDIA GPU resource surfacing in a container for which it was not requested
	nvidiaGpuVisWorkaroundEnvVarName = "NVIDIA_VISIBLE_DEVICES"
	// nvidiaGpuVisWorkaroundEnvVarValue is the value to be set for the environment variable
	// named nvidiaGpuVisWorkaroundEnvVarName, in the above work-around
	nvidiaGpuVisWorkaroundEnvVarValue = "void"
	// defaultBlockDeviceSize is the size for a block volume if it is not specified in the spec
	defaultBlockDeviceSize = "1Gi"
	// blockPvcNamePrefix is the prefix name for the volume device that is auto-created by the statefulset.
	// This is assigned in accordance with the PvcPrefix
	blockPvcNamePrefix = "b"
)

// Streams for stdin, stdout, stderr of executed commands
type Streams struct {
	In     io.Reader
	Out    io.Writer
	ErrOut io.Writer
}

type ArgumentSet struct {
	Logger        logr.Logger
	Cluster       *kdv1.KubeDirectorCluster
	NameSpace     string
	PodName       string
	ContainerID   string
	ContainerName string
}
