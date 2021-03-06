/*
Copyright 2018 The Knative Authors
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

package test

import (
	"github.com/knative/build-pipeline/pkg/apis/pipeline/v1alpha1"
	fakepipelineclientset "github.com/knative/build-pipeline/pkg/client/clientset/versioned/fake"
	informers "github.com/knative/build-pipeline/pkg/client/informers/externalversions"
	informersv1alpha1 "github.com/knative/build-pipeline/pkg/client/informers/externalversions/pipeline/v1alpha1"
	"github.com/knative/build-pipeline/pkg/reconciler"
	"github.com/knative/build-pipeline/pkg/reconciler/v1alpha1/pipelinerun"
	"github.com/knative/build-pipeline/pkg/reconciler/v1alpha1/taskrun"
	buildv1alpha1 "github.com/knative/build/pkg/apis/build/v1alpha1"
	fakebuildclientset "github.com/knative/build/pkg/client/clientset/versioned/fake"
	buildinformers "github.com/knative/build/pkg/client/informers/externalversions"
	buildinformersv1alpha1 "github.com/knative/build/pkg/client/informers/externalversions/build/v1alpha1"
	"github.com/knative/pkg/controller"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"k8s.io/apimachinery/pkg/runtime"
	fakekubeclientset "k8s.io/client-go/kubernetes/fake"
)

// GetLogMessages returns a list of all string logs in logs.
func GetLogMessages(logs *observer.ObservedLogs) []string {
	messages := []string{}
	for _, l := range logs.All() {
		messages = append(messages, l.Message)
	}
	return messages
}

// Data represents the desired state of the system (i.e. existing resources) to seed controllers
// with.
type Data struct {
	PipelineRuns      []*v1alpha1.PipelineRun
	Pipelines         []*v1alpha1.Pipeline
	TaskRuns          []*v1alpha1.TaskRun
	Tasks             []*v1alpha1.Task
	PipelineParams    []*v1alpha1.PipelineParams
	PipelineResources []*v1alpha1.PipelineResource
	Builds            []*buildv1alpha1.Build
}

// Clients holds references to clients which are useful for reconciler tests.
type Clients struct {
	Pipeline *fakepipelineclientset.Clientset
	Build    *fakebuildclientset.Clientset
	Kube     *fakekubeclientset.Clientset
}

// Informers holds references to informers which are useful for reconciler tests.
type Informers struct {
	PipelineRun      informersv1alpha1.PipelineRunInformer
	Pipeline         informersv1alpha1.PipelineInformer
	TaskRun          informersv1alpha1.TaskRunInformer
	Task             informersv1alpha1.TaskInformer
	PipelineParams   informersv1alpha1.PipelineParamsInformer
	PipelineResource informersv1alpha1.PipelineResourceInformer
	Build            buildinformersv1alpha1.BuildInformer
}

func seedTestData(d Data) (Clients, Informers) {
	objs := []runtime.Object{}
	for _, pr := range d.PipelineRuns {
		objs = append(objs, pr)
	}
	for _, p := range d.Pipelines {
		objs = append(objs, p)
	}
	for _, tr := range d.TaskRuns {
		objs = append(objs, tr)
	}
	for _, t := range d.Tasks {
		objs = append(objs, t)
	}
	for _, r := range d.PipelineResources {
		objs = append(objs, r)
	}

	buildObjs := []runtime.Object{}
	for _, b := range d.Builds {
		buildObjs = append(buildObjs, b)
	}
	c := Clients{
		Pipeline: fakepipelineclientset.NewSimpleClientset(objs...),
		Build:    fakebuildclientset.NewSimpleClientset(buildObjs...),
		Kube:     fakekubeclientset.NewSimpleClientset(),
	}
	sharedInformer := informers.NewSharedInformerFactory(c.Pipeline, 0)
	buildInformerFactory := buildinformers.NewSharedInformerFactory(c.Build, 0)

	i := Informers{
		PipelineRun:      sharedInformer.Pipeline().V1alpha1().PipelineRuns(),
		Pipeline:         sharedInformer.Pipeline().V1alpha1().Pipelines(),
		TaskRun:          sharedInformer.Pipeline().V1alpha1().TaskRuns(),
		Task:             sharedInformer.Pipeline().V1alpha1().Tasks(),
		PipelineParams:   sharedInformer.Pipeline().V1alpha1().PipelineParamses(),
		PipelineResource: sharedInformer.Pipeline().V1alpha1().PipelineResources(),
		Build:            buildInformerFactory.Build().V1alpha1().Builds(),
	}

	for _, pr := range d.PipelineRuns {
		i.PipelineRun.Informer().GetIndexer().Add(pr)
	}
	for _, p := range d.Pipelines {
		i.Pipeline.Informer().GetIndexer().Add(p)
	}
	for _, tr := range d.TaskRuns {
		i.TaskRun.Informer().GetIndexer().Add(tr)
	}
	for _, t := range d.Tasks {
		i.Task.Informer().GetIndexer().Add(t)
	}
	for _, t := range d.PipelineParams {
		i.PipelineParams.Informer().GetIndexer().Add(t)
	}
	for _, r := range d.PipelineResources {
		i.PipelineResource.Informer().GetIndexer().Add(r)
	}
	for _, b := range d.Builds {
		i.Build.Informer().GetIndexer().Add(b)
	}
	return c, i
}

// GetTaskRunController returns an instance of the TaskRun controller/reconciler that has been seeded with
// d, where d represents the state of the system (existing resources) needed for the test.
func GetTaskRunController(d Data) (*controller.Impl, *observer.ObservedLogs, Clients) {
	c, i := seedTestData(d)
	observer, logs := observer.New(zap.InfoLevel)
	return taskrun.NewController(
		reconciler.Options{
			Logger:            zap.New(observer).Sugar(),
			KubeClientSet:     c.Kube,
			PipelineClientSet: c.Pipeline,
			BuildClientSet:    c.Build,
		},
		i.TaskRun,
		i.Task,
		i.Build,
		i.PipelineResource,
	), logs, c
}

// GetPipelineRunController returns an instance of the PipelineRun controller/reconciler that has been seeded with
// d, where d represents the state of the system (existing resources) needed for the test.
func GetPipelineRunController(d Data) (*controller.Impl, *observer.ObservedLogs, Clients) {
	c, i := seedTestData(d)
	observer, logs := observer.New(zap.InfoLevel)
	return pipelinerun.NewController(
		reconciler.Options{
			Logger:            zap.New(observer).Sugar(),
			KubeClientSet:     c.Kube,
			PipelineClientSet: c.Pipeline,
		},
		i.PipelineRun,
		i.Pipeline,
		i.Task,
		i.TaskRun,
		i.PipelineParams,
	), logs, c
}
