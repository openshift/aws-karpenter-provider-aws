/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package instancetype

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/samber/lo"

	"github.com/aws/karpenter-provider-aws/pkg/providers/amifamily"
)

// isOpenShiftAMIFamily returns true if the AMIFamily is Custom, which is the only
// AMIFamily OpenShift provisions nodes with.
func isOpenShiftAMIFamily(amiFamily amifamily.AMIFamily) bool {
	_, ok := amiFamily.(*amifamily.Custom)
	return ok
}

// openShiftKubeReservedResources preserves explicit kube-reserved values for Karpenter's
// simulation but doesn't set any defaults.
// OpenShift does not set kube-reserved by default, see https://access.redhat.com/solutions/7127279.
func openShiftKubeReservedResources(kubeReserved map[string]string) corev1.ResourceList {
	return lo.MapEntries(kubeReserved, func(k string, v string) (corev1.ResourceName, resource.Quantity) {
		return corev1.ResourceName(k), resource.MustParse(v)
	})
}

// openShiftSystemReservedResources returns OpenShift's static system-reserved defaults with overrides.
// Defaults source: https://github.com/openshift/machine-config-operator/blob/release-4.22/templates/common/_base/files/kubelet-auto-node-sizing-enabled.yaml
func openShiftSystemReservedResources(systemReserved map[string]string) corev1.ResourceList {
	resources := corev1.ResourceList{
		corev1.ResourceMemory: resource.MustParse("1Gi"),
		corev1.ResourceCPU:    resource.MustParse("500m"),
		// default system-reserved ephemeral-storage, matches SYSTEM_RESERVED_ES in the script
		corev1.ResourceEphemeralStorage: resource.MustParse("1Gi"),
	}
	return lo.Assign(resources, lo.MapEntries(systemReserved, func(k string, v string) (corev1.ResourceName, resource.Quantity) {
		return corev1.ResourceName(k), resource.MustParse(v)
	}))
}
