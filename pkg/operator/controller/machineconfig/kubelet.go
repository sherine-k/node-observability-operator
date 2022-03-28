/*
Copyright 2022.

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

package machineconfigcontroller

import (
	"context"
	"encoding/json"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	//ctrlruntime "sigs.k8s.io/controller-runtime"

	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
)

const (
	// KubeletProfilingConfigName is the name kubelet config CR
	KubeletProfilingConfigName = "99-kubelet-profiling"
)

// ensureKubeletProfConfigExists checks if Kubelet config CR for
// enabling profiling exists, if not creates the resource
func (r *MachineconfigReconciler) ensureKubeletProfConfigExists(ctx context.Context) (*mcv1.KubeletConfig, bool, error) {

	namespace := types.NamespacedName{Name: KubeletProfilingConfigName}
	kubeletmc, exists, err := r.fetchKubeletProfilingConfig(ctx, namespace)
	if err != nil {
		return nil, false, err
	}
	if !exists {
		if err := r.createKubeletProfileConfig(ctx); err != nil {
			return nil, false, err
		}

		kubeletmc, found, err := r.fetchKubeletProfilingConfig(ctx, namespace)
		if err != nil || !found {
			return nil, false, fmt.Errorf("failed to fetch just created kubelet config: %v", err)
		}

		return kubeletmc, true, nil
	}
	return kubeletmc, false, nil
}

// ensurekubeletProfConfigNotExists checks if Kubelet config CR for
// enabling profiling exists, if exists delete the resource
func (r *MachineconfigReconciler) ensureKubeletProfConfigNotExists(ctx context.Context) (bool, error) {

	namespace := types.NamespacedName{Name: KubeletProfilingConfigName}
	kubeletmc, exists, err := r.fetchKubeletProfilingConfig(ctx, namespace)
	if err != nil {
		return false, err
	}

	if exists {
		if err := r.deleteKubeletProfileConfig(ctx, kubeletmc); err != nil {
			return false, err
		}

		_, found, err := r.fetchKubeletProfilingConfig(ctx, namespace)
		if err != nil || found {
			return false, fmt.Errorf("failed to delete kubelet config: %v", err)
		}

		return true, nil
	}
	return false, nil
}

// fetchKubeletProfilingConfig is for fetching the kubelet MC CR created
// by this controller for enabling profiling
func (r *MachineconfigReconciler) fetchKubeletProfilingConfig(ctx context.Context, namespace types.NamespacedName) (*mcv1.KubeletConfig, bool, error) {
	kubeletmc := &mcv1.KubeletConfig{}

	if err := r.Get(ctx, namespace, kubeletmc); err != nil {
		if errors.IsNotFound(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return kubeletmc, true, nil
}

// createKubeletProfileConfig is for creating kubelet MC CR
func (r *MachineconfigReconciler) createKubeletProfileConfig(ctx context.Context) error {
	kubeletmc, err := r.getKubeletConfig()
	if err != nil {
		return err
	}

	if err := r.Create(ctx, kubeletmc); err != nil {
		return fmt.Errorf("failed to create kubelet profiling config %s: %w", kubeletmc.Name, err)
	}
	/*
		if err := ctrlruntime.SetControllerReference(r.CtrlConfig, kubeletmc, r.Scheme); err != nil {
			r.Log.Error(err, "failed to update owner info in profiling KubeletConfig resource")
		}
	*/
	r.Log.Info("successfully created kubelet config for enabling profiling", "KubeletProfilingConfigName", KubeletProfilingConfigName)
	return nil
}

// deleteKubeletProfileConfig is for deleting kubelet MC CR
func (r *MachineconfigReconciler) deleteKubeletProfileConfig(ctx context.Context, kubeletmc *mcv1.KubeletConfig) error {
	if err := r.Delete(ctx, kubeletmc); err != nil {
		return fmt.Errorf("failed to remove kubelet profiling config %s: %w", kubeletmc.Name, err)
	}

	r.Log.Info("successfully removed kubelet config to disable profiling", "KubeletProfilingConfigName", KubeletProfilingConfigName)
	return nil
}

// getKubeletConfig returns the kubelet MC CR data required for creating it
func (r *MachineconfigReconciler) getKubeletConfig() (*mcv1.KubeletConfig, error) {

	config := map[string]bool{
		"enableProfilingHandler": true,
	}

	data, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal kubelet config: %w", err)
	}

	rawExt := &k8sruntime.RawExtension{
		Raw: data,
	}

	return &mcv1.KubeletConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: MCAPIVersion,
			Kind:       "KubeletConfig",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   KubeletProfilingConfigName,
			Labels: MachineConfigLabels,
		},
		Spec: mcv1.KubeletConfigSpec{
			KubeletConfig: rawExt,
			MachineConfigPoolSelector: &metav1.LabelSelector{
				MatchLabels: ProfilingMCSelectorLabels,
			},
		},
	}, nil
}
