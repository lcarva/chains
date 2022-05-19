package objects

import (
	"context"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

// Used as a generic Kubernetes object
// Holds fields common to all objects
// ref: https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.4/pkg/client#Object
type Object interface {
	metav1.Object
	runtime.Object
}

type Result struct {
	Name  string
	Value string
}

// Extending the generic Kubernetes object
// Functions apply to all Tekton objects
type TektonObject interface {
	Object
	GetKind() string
	GetObject() interface{}
	GetLatestAnnotations(ctx context.Context, clientSet versioned.Interface) (map[string]string, error)
	Patch(ctx context.Context, clientSet versioned.Interface, patchBytes []byte) error
	GetResults() []Result
	GetServiceAccountName() string
}

type TaskRunObject struct {
	*v1beta1.TaskRun
}

func NewTaskRunObject(tr *v1beta1.TaskRun) *TaskRunObject {
	return &TaskRunObject{
		tr,
	}
}

func (tro *TaskRunObject) GetKind() string {
	return tro.GetObjectKind().GroupVersionKind().Kind
}

func (tro *TaskRunObject) GetLatestAnnotations(ctx context.Context, clientSet versioned.Interface) (map[string]string, error) {
	tr, err := clientSet.TektonV1beta1().TaskRuns(tro.Namespace).Get(ctx, tro.Name, v1.GetOptions{})
	return tr.Annotations, err
}

func (tro *TaskRunObject) GetObject() interface{} {
	return tro.TaskRun
}

func (tro *TaskRunObject) Patch(ctx context.Context, clientSet versioned.Interface, patchBytes []byte) error {
	_, err := clientSet.TektonV1beta1().TaskRuns(tro.Namespace).Patch(
		ctx, tro.Name, types.MergePatchType, patchBytes, v1.PatchOptions{})
	return err
}

func (tro *TaskRunObject) GetResults() []Result {
	res := []Result{}
	for _, key := range tro.Status.TaskRunResults {
		res = append(res, Result{
			Name:  key.Name,
			Value: key.Value,
		})
	}
	return res
}

func (tro *TaskRunObject) GetServiceAccountName() string {
	return tro.Spec.ServiceAccountName
}

type PipelineRunObject struct {
	*v1beta1.PipelineRun
}

func NewPipelineRunObject(pr *v1beta1.PipelineRun) *PipelineRunObject {
	return &PipelineRunObject{
		pr,
	}
}

func (pro *PipelineRunObject) GetKind() string {
	return pro.GetObjectKind().GroupVersionKind().Kind
}

func (pro *PipelineRunObject) GetLatestAnnotations(ctx context.Context, clientSet versioned.Interface) (map[string]string, error) {
	pr, err := clientSet.TektonV1beta1().PipelineRuns(pro.Namespace).Get(ctx, pro.Name, v1.GetOptions{})
	return pr.Annotations, err
}

func (pro *PipelineRunObject) GetObject() interface{} {
	return pro.PipelineRun
}

func (pro *PipelineRunObject) Patch(ctx context.Context, clientSet versioned.Interface, patchBytes []byte) error {
	_, err := clientSet.TektonV1beta1().PipelineRuns(pro.Namespace).Patch(
		ctx, pro.Name, types.MergePatchType, patchBytes, v1.PatchOptions{})
	return err
}

func (pro *PipelineRunObject) GetResults() []Result {
	res := []Result{}
	for _, key := range pro.Status.PipelineResults {
		res = append(res, Result{
			Name:  key.Name,
			Value: key.Value,
		})
	}
	return res
}

func (pro *PipelineRunObject) GetServiceAccountName() string {
	return pro.Spec.ServiceAccountName
}
