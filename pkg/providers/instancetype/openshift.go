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
	"math"

	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
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

// openShiftSystemReservedResources returns OpenShift system-reserved defaults with overrides.
// Formula source: https://github.com/openshift/machine-config-operator/blob/master/templates/common/_base/files/kubelet-auto-sizing.yaml
func openShiftSystemReservedResources(mem *resource.Quantity, info ec2types.InstanceTypeInfo, systemReserved map[string]string) corev1.ResourceList {
	resources := corev1.ResourceList{
		corev1.ResourceMemory: openShiftSystemReservedMemory(mem),
		corev1.ResourceCPU:    openShiftSystemReservedCPU(cpu(info)),
		// default system-reserved ephemeral-storage, matches SYSTEM_RESERVED_ES in the script
		corev1.ResourceEphemeralStorage: resource.MustParse("1Gi"),
	}
	return lo.Assign(resources, lo.MapEntries(systemReserved, func(k string, v string) (corev1.ResourceName, resource.Quantity) {
		return corev1.ResourceName(k), resource.MustParse(v)
	}))
}

// openShiftSystemReservedMemory calculates OpenShift's dynamic memory reservation.
func openShiftSystemReservedMemory(mem *resource.Quantity) resource.Quantity {
	totalGiB := float64(mem.Value() / (1024 * 1024 * 1024))
	reservedGiB := 1.0
	if totalGiB <= 8 {
		totalGiB = 0
	} else {
		reservedGiB = 1
		totalGiB -= 8
	}
	if totalGiB <= 120 {
		reservedGiB += totalGiB * 0.06
		totalGiB = 0
	} else {
		reservedGiB += 6.72
		totalGiB -= 112
	}
	if totalGiB >= 0 {
		reservedGiB += totalGiB * 0.02
	}
	return *resource.NewQuantity(int64(math.Round(reservedGiB))*1024*1024*1024, resource.BinarySI)
}

// openShiftSystemReservedCPU calculates OpenShift's dynamic CPU reservation.
func openShiftSystemReservedCPU(cpus *resource.Quantity) resource.Quantity {
	vCPU := float64(cpus.Value())
	reservedCores := 0.06
	if vCPU > 1 {
		reservedCores += 0.012 * (vCPU - 1)
	}
	reservedCores = math.Round(reservedCores*100) / 100
	reservedMilli := int64(math.Round(reservedCores * 1000))
	if reservedMilli < 500 {
		reservedMilli = 500
	}
	return *resource.NewMilliQuantity(reservedMilli, resource.DecimalSI)
}
