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
	"context"
	"fmt"

	"github.com/asteris-llc/converge/graph"
	"github.com/asteris-llc/converge/render"
)

// PipelinePlan runs plans on the graph based on pipeline generation
func PipelinePlan(ctx context.Context, in *graph.Graph) (*graph.Graph, error) {
	var hasErrors error

	renderingPlant, err := render.NewFactory(ctx, in)
	if err != nil {
		return nil, err
	}

	out, err := in.Transform(ctx, func(id string, out *graph.Graph) error {
		planner := Pipeline(out, id, renderingPlant)
		result := planner.Execute()
		val, isRight := result.FromEither()
		if !isRight {
			return fmt.Errorf("%v", val)
		}
		return nil
	})

	if err != nil {
		return out, err
	}

	return out, hasErrors
}
