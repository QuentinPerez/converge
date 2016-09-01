// Copyright © 2016 Asteris, LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package plan

import (
	"errors"
	"fmt"

	"github.com/asteris-llc/converge/executor"
	"github.com/asteris-llc/converge/executor/either"
	"github.com/asteris-llc/converge/graph"
	"github.com/asteris-llc/converge/render"
	"github.com/asteris-llc/converge/resource"
)

type pipelineGen struct {
	Graph          *graph.Graph
	RenderingPlant *render.Factory
}

type taskWrapper struct {
	ID   string
	Task resource.Task
}

// Pipeline generates a pipeline to evaluate a single graph node
func Pipeline(g *graph.Graph, id string, factory *render.Factory) executor.Pipeline {
	gen := pipelineGen{Graph: g, RenderingPlant: factory}
	return executor.NewPipeline().
		AndThen(gen.GetTask).
		AndThen(gen.DependencyCheck).
		AndThen(gen.PlanNode)
}

// GetTask returns Right Task if the value is a task, or Left Error if not
func (g pipelineGen) GetTask(idi interface{}) either.EitherM {
	id := idi.(string)
	node := g.Graph.Get(id)
	if task, ok := node.(resource.Task); ok {
		return either.RightM(taskWrapper{ID: id, Task: task})
	}
	return either.LeftM(fmt.Errorf("expected resource.Task but got %T", node))
}

// DependencyCheck looks for failing dependency nodes.  If an error is
// encountered it returns `Left error`, if failing dependencies are encountered
// it returns `Right (Left Status)` and otherwise returns `Right (Right
// Task)`. The return values are structured to short-circuit `PlanNode` if we
// have failures.
func (g pipelineGen) DependencyCheck(taskI interface{}) either.EitherM {
	task, ok := taskI.(taskWrapper)
	if !ok {
		return either.LeftM(errors.New("input node is not a task wrapper"))
	}
	for _, depID := range graph.Targets(g.Graph.DownEdges(task.ID)) {
		dep, ok := g.Graph.Get(depID).(executor.Status)
		if !ok {
			return either.LeftM(errors.New("dependency is not a status node"))
		}
		if err := dep.Error(); err != nil {
			errResult := &Result{
				Status: &resource.Status{WillChange: true},
				Task:   task.Task,
				Err:    fmt.Errorf("error in dependency %q", depID),
			}
			return either.RightM(either.LeftM(errResult))
		}
	}
	return either.RightM(either.RightM(task))
}

// PlanNode runs plan on the node, it takes an Either *Result TaskWrapper and,
// if the input value is Left, returns it as a Right value, otherwise it
// attempts to run plan on the TaskWrapper and returns an appropriate Left or
// Right value.
func (g pipelineGen) PlanNode(taski interface{}) either.EitherM {
	taskE, ok := taski.(either.EitherM)
	if !ok {
		return either.LeftM(errors.New("plan node was expected to be EitherM"))
	}
	val, isRight := taskE.FromEither()
	if !isRight {
		return either.RightM(val)
	}
	twrapper, ok := val.(taskWrapper)
	if !ok {
		return either.LeftM(fmt.Errorf("plan expected a taskWrapper but got %T", val))
	}
	renderer, err := g.Renderer(twrapper.ID)
	if err != nil {
		return either.LeftM(fmt.Errorf("unable to get renderer for %s", twrapper.ID))
	}
	status, err := twrapper.Task.Check(renderer)
	return either.RightM(&Result{
		Status: status,
		Task:   twrapper.Task,
		Err:    err,
	})
}

func (g pipelineGen) Renderer(id string) (*render.Renderer, error) {
	return g.RenderingPlant.GetRenderer(id)
}
