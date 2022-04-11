package objects

import (
	"context"
	"fmt"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type Annotation struct {
	Err   error
	Value string
	Ok    bool
}

// Represents a generic K8s object
// This isn't meant to be a final implementation, just one approach
// Commented out methods will be required for other storage mechanisms,
//  but we're focused on Tekton for now
type K8sObject interface {
	GetName() string
	GetNamespace() string
	GetKind() string
	GetAnnotation(annotation string) *Annotation
	GetLatestAnnotation(annotation string) *Annotation
	GetObject() interface{}
	Patch(patchBytes []byte) error
	// GetUID() string
	// GetGroupVersion() string
	// GetStatus() interface{}
}

type TaskRunObject struct {
	tr        *v1beta1.TaskRun
	clientSet versioned.Interface
	ctx       context.Context
}

func NewTaskRunObject(tr *v1beta1.TaskRun, clientSet versioned.Interface, ctx context.Context) *TaskRunObject {
	return &TaskRunObject{
		tr:        tr,
		clientSet: clientSet,
		ctx:       ctx,
	}
}

func (tro *TaskRunObject) GetName() string {
	return tro.tr.Name
}

func (tro *TaskRunObject) GetNamespace() string {
	return tro.tr.Namespace
}

func (tro *TaskRunObject) GetKind() string {
	return tro.tr.Kind
}

func (tro *TaskRunObject) GetAnnotation(annotation string) *Annotation {
	val, ok := tro.tr.Annotations[annotation]
	return &Annotation{
		Err:   nil,
		Value: val,
		Ok:    ok,
	}
}

func (tro *TaskRunObject) GetLatestAnnotation(annotation string) *Annotation {
	tr, err := tro.clientSet.TektonV1beta1().TaskRuns(tro.tr.Namespace).Get(tro.ctx, tro.tr.Name, v1.GetOptions{})
	if err != nil {
		return &Annotation{
			Err:   fmt.Errorf("error retrieving taskrun: %s", err),
			Value: "",
			Ok:    false,
		}
	}
	val, ok := tr.Annotations[annotation]
	return &Annotation{
		Err:   nil,
		Value: val,
		Ok:    ok,
	}
}

func (tro *TaskRunObject) GetObject() interface{} {
	return tro.tr
}

func (tro *TaskRunObject) Patch(patchBytes []byte) error {
	_, err := tro.clientSet.TektonV1beta1().TaskRuns(tro.tr.Namespace).Patch(
		tro.ctx, tro.tr.Name, types.MergePatchType, patchBytes, v1.PatchOptions{})
	return err
}

type PipelineRunObject struct {
	pr        *v1beta1.PipelineRun
	clientSet versioned.Interface
	ctx       context.Context
}

func NewPipelineRunObject(pr *v1beta1.PipelineRun, clientSet versioned.Interface, ctx context.Context) *PipelineRunObject {
	return &PipelineRunObject{
		pr:        pr,
		clientSet: clientSet,
		ctx:       ctx,
	}
}

func (pro *PipelineRunObject) GetName() string {
	return pro.pr.Name
}

func (pro *PipelineRunObject) GetNamespace() string {
	return pro.pr.Namespace
}

func (pro *PipelineRunObject) GetKind() string {
	return pro.pr.Kind
}

func (pro *PipelineRunObject) GetAnnotation(annotation string) *Annotation {
	val, ok := pro.pr.Annotations[annotation]
	return &Annotation{
		Err:   nil,
		Value: val,
		Ok:    ok,
	}
}

func (pro *PipelineRunObject) GetLatestAnnotation(annotation string) *Annotation {
	tr, err := pro.clientSet.TektonV1beta1().PipelineRuns(pro.pr.Namespace).Get(pro.ctx, pro.pr.Name, v1.GetOptions{})
	if err != nil {
		return &Annotation{
			Err:   fmt.Errorf("error retrieving pipelinerun: %s", err),
			Value: "",
			Ok:    false,
		}
	}
	val, ok := tr.Annotations[annotation]
	return &Annotation{
		Err:   nil,
		Value: val,
		Ok:    ok,
	}
}

func (pro *PipelineRunObject) GetObject() interface{} {
	return pro.pr
}

func (pro *PipelineRunObject) Patch(patchBytes []byte) error {
	_, err := pro.clientSet.TektonV1beta1().PipelineRuns(pro.pr.Namespace).Patch(
		pro.ctx, pro.pr.Name, types.MergePatchType, patchBytes, v1.PatchOptions{})
	return err
}
