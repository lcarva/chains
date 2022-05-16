/*
Copyright 2021 The Tekton Authors

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

package taskrun

import (
	"github.com/tektoncd/chains/pkg/chains/formats/intotoite6/util"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

// BuildConfig is the custom Chains format to fill out the
// "buildConfig" section of the slsa-provenance predicate
type BuildConfig struct {
	Steps []util.StepAttestation `json:"steps"`
}

// Step corresponds to one step in the TaskRun
type Step struct {
	EntryPoint  string            `json:"entryPoint"`
	Arguments   interface{}       `json:"arguments,omitempty"`
	Environment interface{}       `json:"environment,omitempty"`
	Annotations map[string]string `json:"annotations"`
}

func buildConfig(tr *v1beta1.TaskRun) BuildConfig {
	attestations := []util.StepAttestation{}
	for _, stepState := range tr.Status.Steps {
		step := stepFromTaskRun(stepState.Name, tr)
		attestations = append(attestations, util.AttestStep(step, &stepState))
	}
	return BuildConfig{Steps: attestations}
}

func stepFromTaskRun(name string, tr *v1beta1.TaskRun) *v1beta1.Step {
	if tr.Status.TaskSpec != nil {
		for _, s := range tr.Status.TaskSpec.Steps {
			if s.Name == name {
				return &s
			}
		}
	}
	return &v1beta1.Step{}
}
