package util

import (
	"fmt"
	"sort"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	slsa "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v0.2"
	"github.com/tektoncd/chains/pkg/artifacts"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"go.uber.org/zap"

	intoto "github.com/in-toto/in-toto-golang/in_toto"
)

const (
	TektonID                     = "https://tekton.dev/attestations/chains@v2"
	TektonPipelineRunID          = "https://tekton.dev/attestations/chains/pipelinerun@v2"
	CommitParam                  = "CHAINS-GIT_COMMIT"
	UrlParam                     = "CHAINS-GIT_URL"
	ChainsReproducibleAnnotation = "chains.tekton.dev/reproducible"
)

// GetSubjectDigests extracts OCI images from the TaskRun based on standard hinting set up
// It also goes through looking for any PipelineResources of Image type
func GetSubjectDigests(obj objects.TektonObject, logger *zap.SugaredLogger) []intoto.Subject {
	var subjects []intoto.Subject
	imgs := artifacts.ExtractOCIImagesFromResults(obj, logger)
	for _, i := range imgs {
		if d, ok := i.(name.Digest); ok {
			subjects = append(subjects, intoto.Subject{
				Name: d.Repository.Name(),
				Digest: slsa.DigestSet{
					"sha256": strings.TrimPrefix(d.DigestStr(), "sha256:"),
				},
			})
		}
	}

	// Check if object is a Taskrun, if so search for images used in PipelineResources
	// Otherwise object is a PipelineRun, where Pipelineresources are not relevant.
	// PipelineResources have been deprecated so their support has been left out of
	// the POC for TEP-84
	// More info: https://tekton.dev/docs/pipelines/resources/
	tr, ok := obj.GetObject().(*v1beta1.TaskRun)
	if !ok || tr.Spec.Resources == nil {
		return subjects
	}

	// go through resourcesResult
	for _, output := range tr.Spec.Resources.Outputs {
		name := output.Name
		if output.PipelineResourceBinding.ResourceSpec == nil {
			continue
		}
		// similarly, we could do this for other pipeline resources or whatever thing replaces them
		if output.PipelineResourceBinding.ResourceSpec.Type == v1alpha1.PipelineResourceTypeImage {
			// get the url and digest, and save as a subject
			var url, digest string
			for _, s := range tr.Status.ResourcesResult {
				if s.ResourceName == name {
					if s.Key == "url" {
						url = s.Value
					}
					if s.Key == "digest" {
						digest = s.Value
					}
				}
			}
			subjects = append(subjects, intoto.Subject{
				Name: url,
				Digest: slsa.DigestSet{
					"sha256": strings.TrimPrefix(digest, "sha256:"),
				},
			})
		}
	}
	sort.Slice(subjects, func(i, j int) bool {
		return subjects[i].Name <= subjects[j].Name
	})
	return subjects
}

// supports the SPDX format which is recommended by in-toto
// ref: https://spdx.dev/spdx-specification-21-web-version/#h.49x2ik5
// ref: https://github.com/in-toto/attestation/blob/849867bee97e33678f61cc6bd5da293097f84c25/spec/field_types.md
func SpdxGit(url, revision string) string {
	prefix := "git+"
	if revision == "" {
		return prefix + url + ".git"
	}
	return prefix + url + fmt.Sprintf("@%s", revision)
}

type StepAttestation struct {
	EntryPoint  string            `json:"entryPoint"`
	Arguments   interface{}       `json:"arguments,omitempty"`
	Environment interface{}       `json:"environment,omitempty"`
	Annotations map[string]string `json:"annotations"`
}

func AttestStep(step *v1beta1.Step, stepState *v1beta1.StepState) StepAttestation {
	attestation := StepAttestation{}

	entrypoint := strings.Join(step.Command, " ")
	if step.Script != "" {
		entrypoint = step.Script
	}
	attestation.EntryPoint = entrypoint
	attestation.Arguments = step.Args

	env := map[string]interface{}{}
	env["image"] = stepState.ImageID
	env["container"] = stepState.Name
	attestation.Environment = env

	return attestation
}

func AttestInvocation(params []v1beta1.Param, paramSpecs []v1beta1.ParamSpec, logger *zap.SugaredLogger) slsa.ProvenanceInvocation {
	i := slsa.ProvenanceInvocation{}
	iParams := make(map[string]string)

	// get implicit parameters from defaults
	for _, p := range paramSpecs {
		if p.Default != nil {
			v, err := p.Default.MarshalJSON()
			if err != nil {
				logger.Errorf("Unable to marshall %q default parameter: %s", p, err)
				continue
			}
			iParams[p.Name] = string(v)
		}
	}

	// get explicit parameters
	for _, p := range params {
		v, err := p.Value.MarshalJSON()
		if err != nil {
			logger.Errorf("Unable to marshall %q parameter: %s", p, err)
			continue
		}
		iParams[p.Name] = string(v)
	}

	i.Parameters = iParams
	return i
}
