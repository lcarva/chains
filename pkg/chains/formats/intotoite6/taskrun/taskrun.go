package taskrun

import (
	intoto "github.com/in-toto/in-toto-golang/in_toto"
	slsa "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v0.2"
	"github.com/tektoncd/chains/pkg/chains/formats/intotoite6/util"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"go.uber.org/zap"
)

func GenerateAttestation(builderID string, tr *v1beta1.TaskRun, logger *zap.SugaredLogger) (interface{}, error) {
	tro := objects.NewTaskRunObject(tr)
	subjects := util.GetSubjectDigests(tro, logger)

	att := intoto.ProvenanceStatement{
		StatementHeader: intoto.StatementHeader{
			Type:          intoto.StatementInTotoV01,
			PredicateType: slsa.PredicateSLSAProvenance,
			Subject:       subjects,
		},
		Predicate: slsa.ProvenancePredicate{
			Builder: slsa.ProvenanceBuilder{
				ID: builderID,
			},
			BuildType:   util.TektonID,
			Invocation:  invocation(tr),
			BuildConfig: buildConfig(tr),
			Metadata:    metadata(tr),
			Materials:   materials(tr),
		},
	}
	return att, nil
}

// invocation describes the event that kicked off the build
// we currently don't set ConfigSource because we don't know
// which material the Task definition came from
func invocation(tr *v1beta1.TaskRun) slsa.ProvenanceInvocation {
	var paramSpecs []v1beta1.ParamSpec
	if ts := tr.Status.TaskSpec; ts != nil {
		paramSpecs = ts.Params
	}
	return util.AttestInvocation(tr.Spec.Params, paramSpecs)
}

func metadata(tr *v1beta1.TaskRun) *slsa.ProvenanceMetadata {
	m := &slsa.ProvenanceMetadata{}
	if tr.Status.StartTime != nil {
		m.BuildStartedOn = &tr.Status.StartTime.Time
	}
	if tr.Status.CompletionTime != nil {
		m.BuildFinishedOn = &tr.Status.CompletionTime.Time
	}
	for label, value := range tr.Labels {
		if label == util.ChainsReproducibleAnnotation && value == "true" {
			m.Reproducible = true
		}
	}
	return m
}

// add any Git specification to materials
func materials(tr *v1beta1.TaskRun) []slsa.ProvenanceMaterial {
	var mats []slsa.ProvenanceMaterial
	gitCommit, gitURL := gitInfo(tr)

	// Store git rev as Materials and Recipe.Material
	if gitCommit != "" && gitURL != "" {
		mats = append(mats, slsa.ProvenanceMaterial{
			URI:    gitURL,
			Digest: map[string]string{"sha1": gitCommit},
		})
		return mats
	}

	if tr.Spec.Resources == nil {
		return mats
	}

	// check for a Git PipelineResource
	for _, input := range tr.Spec.Resources.Inputs {
		if input.ResourceSpec == nil || input.ResourceSpec.Type != v1alpha1.PipelineResourceTypeGit {
			continue
		}

		m := slsa.ProvenanceMaterial{
			Digest: slsa.DigestSet{},
		}

		for _, rr := range tr.Status.ResourcesResult {
			if rr.ResourceName != input.Name {
				continue
			}
			if rr.Key == "url" {
				m.URI = util.SpdxGit(rr.Value, "")
			} else if rr.Key == "commit" {
				m.Digest["sha1"] = rr.Value
			}
		}

		var url string
		var revision string
		for _, param := range input.ResourceSpec.Params {
			if param.Name == "url" {
				url = param.Value
			}
			if param.Name == "revision" {
				revision = param.Value
			}
		}
		m.URI = util.SpdxGit(url, revision)
		mats = append(mats, m)
	}
	return mats
}

// gitInfo scans over the input parameters and looks for parameters
// with specified names.
func gitInfo(tr *v1beta1.TaskRun) (commit string, url string) {
	// Scan for git params to use for materials
	for _, p := range tr.Spec.Params {
		if p.Name == util.CommitParam {
			commit = p.Value.StringVal
			continue
		}
		if p.Name == util.UrlParam {
			url = p.Value.StringVal
		}
	}

	if tr.Status.TaskSpec != nil {
		for _, p := range tr.Status.TaskSpec.Params {
			if p.Default == nil {
				continue
			}
			if p.Name == util.CommitParam {
				commit = p.Default.StringVal
				continue
			}
			if p.Name == util.UrlParam {
				url = p.Default.StringVal
			}
		}
	}

	for _, r := range tr.Status.TaskRunResults {
		if r.Name == util.CommitParam {
			commit = r.Value
		}
		if r.Name == util.UrlParam {
			url = r.Value
		}
	}

	url = util.SpdxGit(url, "")
	return
}
