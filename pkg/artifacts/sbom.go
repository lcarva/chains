/*
Copyright 2023 The Tekton Authors
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

package artifacts

import (
	"context"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/config"
	"k8s.io/apimachinery/pkg/util/sets"
	"knative.dev/pkg/logging"
)

type SBOMArtifact struct{}

var _ Signable = &SBOMArtifact{}

func (sa *SBOMArtifact) ExtractObjects(ctx context.Context, tektonObject objects.TektonObject) []interface{} {
	var objs []interface{}
	for _, obj := range extractSBOMFromResults(ctx, tektonObject) {
		objs = append(objs, objects.NewSBOMObject(obj.SBOMURI, obj.SBOMFormat, obj.URI, obj.Digest, tektonObject))
	}
	return objs
}

func (sa *SBOMArtifact) StorageBackend(cfg config.Config) sets.Set[string] {
	return cfg.Artifacts.SBOM.StorageBackend
}
func (sa *SBOMArtifact) Signer(cfg config.Config) string {
	return cfg.Artifacts.SBOM.Signer
}
func (sa *SBOMArtifact) PayloadFormat(cfg config.Config) config.PayloadType {
	return config.PayloadType(cfg.Artifacts.SBOM.Format)
}

func (sa *SBOMArtifact) FullKey(obj interface{}) string {
	v := obj.(*objects.SBOMObject)
	return v.GetSBOMURL()
}

func (sa *SBOMArtifact) ShortKey(obj interface{}) string {
	v := obj.(*objects.SBOMObject)
	return v.GetSBOMURL()
}
func (sa *SBOMArtifact) Type() string {
	return "sbom"
}
func (sa *SBOMArtifact) Enabled(cfg config.Config) bool {
	return cfg.Artifacts.SBOM.Enabled()
}

func hasSBOMRequirements(s StructuredSignable) bool {
	return s.Digest != "" && s.URI != "" && s.SBOMURI != "" && s.SBOMFormat != ""
}

func extractSBOMFromResults(ctx context.Context, tektonObject objects.TektonObject) []StructuredSignable {
	logger := logging.FromContext(ctx)
	var objs []StructuredSignable

	sse := structuredSignableExtractor{
		uriSuffix:        "IMAGE_URL",
		digestSuffix:     "IMAGE_DIGEST",
		sbomURISuffix:    "IMAGE_SBOM_URL",
		sbomFormatSuffix: "IMAGE_SBOM_FORMAT",
		isValid:          hasSBOMRequirements,
	}
	for _, s := range sse.extract(ctx, tektonObject) {
		if _, err := name.NewDigest(s.SBOMURI); err != nil {
			logger.Errorf("error getting digest for SBOM image %s: %v", s.SBOMURI, err)
			continue
		}
		objs = append(objs, s)
	}

	var images []name.Digest
	var sboms []string
	var sbomsFormat string
	// look for a comma separated list of images and their SBOMs
	for _, key := range tektonObject.GetResults() {
		switch key.Name {
		case "IMAGES":
			for _, img := range strings.FieldsFunc(key.Value.StringVal, split) {
				trimmed := strings.TrimSpace(img)
				if trimmed == "" {
					continue
				}

				dgst, err := name.NewDigest(trimmed)
				if err != nil {
					logger.Errorf("error getting digest for img %s: %v", trimmed, err)
					continue
				}
				images = append(images, dgst)
			}
		case "SBOMS":
			for _, img := range strings.FieldsFunc(key.Value.StringVal, split) {
				trimmed := strings.TrimSpace(img)
				if trimmed == "" {
					continue
				}
				if _, err := name.NewDigest(trimmed); err != nil {
					logger.Errorf("error getting digest for SBOM image %s: %v", trimmed, err)
					continue
				}
				sboms = append(sboms, trimmed)
			}
		case "SBOMS_FORMAT":
			f := strings.TrimSpace(key.Value.StringVal)
			if f != "" {
				sbomsFormat = f
			}
		}
	}

	if len(images) != len(sboms) {
		logger.Warnf("IMAGES and SBOMS do not contain the same amount of entries")
		return objs
	}

	if len(sboms) > 0 && sbomsFormat == "" {
		logger.Warnf("SBOMS_FORMAT not specified")
		return objs
	}

	for i, sbom := range sboms {
		img := images[i]
		objs = append(objs, StructuredSignable{
			URI:        img.Name(),
			Digest:     img.Identifier(),
			SBOMURI:    sbom,
			SBOMFormat: sbomsFormat,
		})
	}
	return objs
}
