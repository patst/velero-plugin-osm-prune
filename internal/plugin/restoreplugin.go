/*
Copyright 2018, 2019 the Velero contributors.

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

package plugin

import (
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// RestorePlugin is a restore item action plugin for Velero
type RestorePlugin struct {
	log logrus.FieldLogger
}

// NewRestorePlugin instantiates a RestorePlugin.
func NewRestorePlugin(log logrus.FieldLogger) *RestorePlugin {
	return &RestorePlugin{log: log}
}

// AppliesTo returns information about which resources this action should be invoked for.
// The IncludedResources and ExcludedResources slices can include both resources
// and resources with group names. These work: "ingresses", "ingresses.extensions".
// A RestoreItemAction's Execute function will only be invoked on items that match the returned
// selector. A zero-valued ResourceSelector matches all resources.
func (p *RestorePlugin) AppliesTo() (velero.ResourceSelector, error) {
	return velero.ResourceSelector{
		IncludedResources: []string{"pods"},
	}, nil
}

// Execute allows the RestorePlugin to perform arbitrary logic with the item being restored,
// in this case, setting a custom annotation on the item being restored.
func (p *RestorePlugin) Execute(input *velero.RestoreItemActionExecuteInput) (*velero.RestoreItemActionExecuteOutput, error) {
	p.log.Infof("Executing OSM-Prune plugin for Restore %s", input.Restore.Name)

	metadata, err := meta.Accessor(input.Item)
	if err != nil {
		return &velero.RestoreItemActionExecuteOutput{}, err
	}

	pod := new(v1.Pod)
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(input.Item.UnstructuredContent(), pod); err != nil {
		p.log.Error("Error converting item to pod schema", err)
		return &velero.RestoreItemActionExecuteOutput{}, errors.WithStack(err)
	}

	// Check if the pod was managed by OSM (e.g. there is a osm-proxy-uuid label)
	labels := metadata.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	if _, ok := labels["osm-proxy-uuid"]; ok {
		p.log.Infof("Found osm-proxy-uuid label on pod %s, removing osm-init and envoy (init-)containers", pod.Name)
		// Remove the container named osm-init (if existing)
		// .spec.initContainers[name=osm-init]
		if pod.Spec.InitContainers != nil {
			for idx, c := range pod.Spec.InitContainers {
				if c.Name == "osm-init" {
					p.log.Infof("Removed osm-init container from pod %s", pod.Name)
					pod.Spec.InitContainers = removeIndexFromSlice(pod.Spec.InitContainers, idx)
				}
			}
		}

		// Remove the sidecar (if existing)
		// .spec.containers[name=envoy]
		if pod.Spec.Containers != nil {
			for idx, c := range pod.Spec.Containers {
				if c.Name == "envoy" {
					p.log.Infof("Removed envoy container from pod %s", pod.Name)
					pod.Spec.Containers = removeIndexFromSlice(pod.Spec.Containers, idx)
				}
			}
			// remove the envoy volume
			// .spec.volumes[name=envoy-bootstrap-config-volume]
			for idx, v := range pod.Spec.Volumes {
				if v.Name == "envoy-bootstrap-config-volume" {
					p.log.Infof("Removed envoy-bootstrap-config-volume from pod %s", pod.Name)
					pod.Spec.Volumes = append(pod.Spec.Volumes[:idx], pod.Spec.Volumes[idx+1:]...)
				}
			}
		}

		// convert back and return the mapped result
		res, err := runtime.DefaultUnstructuredConverter.ToUnstructured(pod)
		if err != nil {
			p.log.Errorf("Error converting item back to unstructured schema")
			return &velero.RestoreItemActionExecuteOutput{}, errors.WithStack(err)
		}
		return velero.NewRestoreItemActionExecuteOutput(&unstructured.Unstructured{Object: res}), nil

	} else {
		p.log.Infof("Found no osm-proxy-uuid label on pod %s, not relevant for osm-prune-plugin", pod.Name)
	}
	return velero.NewRestoreItemActionExecuteOutput(input.Item), nil
}

func removeIndexFromSlice(s []v1.Container, index int) []v1.Container {
	return append(s[:index], s[index+1:]...)
}
