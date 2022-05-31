/*
Copyright 2020 The Tekton Authors
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

package pipelinerun

import (
	"context"

	signing "github.com/tektoncd/chains/pkg/chains"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	pipelinerunreconciler "github.com/tektoncd/pipeline/pkg/client/injection/reconciler/pipeline/v1beta1/pipelinerun"
	"knative.dev/pkg/logging"
	pkgreconciler "knative.dev/pkg/reconciler"
)

const (
	// SecretPath contains the path to the secrets volume that is mounted in.
	SecretPath = "/etc/signing-secrets"
)

type Reconciler struct {
	PipelineRunSigner signing.Signer
}

// Check that our Reconciler implements pipelinerunreconciler.Interface and pipelinerunreconciler.Finalizer
var _ pipelinerunreconciler.Interface = (*Reconciler)(nil)
var _ pipelinerunreconciler.Finalizer = (*Reconciler)(nil)

// ReconcileKind  handles a changed or created PipelineRun.
// This is the main entrypoint for chains business logic.
func (r *Reconciler) ReconcileKind(ctx context.Context, pr *v1beta1.PipelineRun) pkgreconciler.Event {
	return r.FinalizeKind(ctx, pr)
}

// FinalizeKind implements pipelinerunreconciler.Finalizer
// We utilize finalizers to ensure that we get a crack at signing every pipelinerun
// that we see flowing through the system.  If we don't add a finalizer, it could
// get cleaned up before we see the final state and sign it.
func (r *Reconciler) FinalizeKind(ctx context.Context, pr *v1beta1.PipelineRun) pkgreconciler.Event {
	// Check to make sure the PipelineRun is finished.
	if !pr.IsDone() {
		logging.FromContext(ctx).Infof("pipelinerun %s/%s is still running", pr.Namespace, pr.Name)
		return nil
	}
	pro := objects.NewPipelineRunObject(pr)

	// Check to see if it has already been signed.
	if signing.Reconciled(pro) {
		logging.FromContext(ctx).Infof("pipelinerun %s/%s has been reconciled", pr.Namespace, pr.Name)
		return nil
	}

	// TaskRuns within a PipelineRun may not have been finalized yet if the PipelineRun timeout
	// has exceeded. Wait to process the PipelineRun on the next update, see
	// https://github.com/tektoncd/pipeline/issues/4916
	for _, tr := range pr.Status.TaskRuns {
		if tr.Status == nil || tr.Status.CompletionTime == nil {
			logging.FromContext(ctx).Infof(
				"taskrun %s within pipelinerun %s/%s is not yet finalized",
				tr.PipelineTaskName, pr.Namespace, pr.Name)
			return nil
		}
	}

	if err := r.PipelineRunSigner.Sign(ctx, pro); err != nil {
		return err
	}
	return nil
}
