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

package providers

import (
	"fmt"
	"strings"

	pp "github.com/asteris-llc/converge/prettyprinters"
	"github.com/asteris-llc/converge/prettyprinters/graphviz"
	"github.com/asteris-llc/converge/resource/file/content"
	"github.com/asteris-llc/converge/resource/module"
	"github.com/asteris-llc/converge/resource/param"
	"github.com/asteris-llc/converge/resource/shell"
)

// PreparerProvider is the PrintProvider type for Preparer resources
type PreparerProvider struct {
	graphviz.GraphIDProvider
	ShowParams bool
}

// VertexGetID returns a the Graph ID as the VertexID, possibly masking it
// depending on the vertex type and configuration
func (p PreparerProvider) VertexGetID(e graphviz.GraphEntity) (pp.VisibleRenderable, error) {
	switch e.Value.(type) {
	case *param.Preparer:
		return pp.RenderableString(e.Name, p.ShowParams), nil
	default:
		return pp.VisibleString(e.Name), nil
	}
}

// VertexGetLabel returns a vertex label based on the type of the preparer. The
// specific generated labels are:
//   Templates: Return 'Template' and the file destination
//   Modules: Return 'Module' and the module name
//   Params:  Return 'name -> "value"'
//   otherwise: Return 'name'
func (p PreparerProvider) VertexGetLabel(e graphviz.GraphEntity) (pp.VisibleRenderable, error) {
	var name string

	if e.Name == rootNodeID {
		name = "/"
	} else {
		name = strings.Split(e.Name, "root/")[1]
	}

	switch e.Value.(type) {
	case *content.Preparer:
		v := e.Value.(*content.Preparer)
		return pp.VisibleString(fmt.Sprintf("File: %s", v.Destination)), nil
	case *module.Preparer:
		return pp.VisibleString(fmt.Sprintf("Module: %s", name)), nil
	case *param.Preparer:
		v := e.Value.(*param.Preparer)
		var paramStr string
		if v.Default == nil {
			paramStr = fmt.Sprintf(`%s = <required param>`, name)
		} else {
			paramStr = fmt.Sprintf(`%s = \"%s\"`, name, *v.Default)
		}
		return pp.RenderableString(paramStr, p.ShowParams), nil
	default:
		return pp.VisibleString(name), nil
	}
}

// VertexGetProperties sets graphviz attributes based on the type of the
// preparer. Specifically, we set the shape to 'component' for Shell preparers
// and 'tab' for templates, and we set the entire root node to be invisible.
func (p PreparerProvider) VertexGetProperties(e graphviz.GraphEntity) graphviz.PropertySet {
	properties := make(map[string]string)
	switch e.Value.(type) {
	case *shell.Preparer:
		properties["shape"] = "component"
	case *content.Preparer:
		properties["shape"] = "tab"
	case *param.Preparer:
		v := e.Value.(*param.Preparer)
		if v.Default == nil {
			properties["style"] = "dotted"
		}
	}
	return properties
}

// EdgeGetProperties sets attributes for graph edges, specifically making edges
// originating from the Root node invisible.
func (p PreparerProvider) EdgeGetProperties(src graphviz.GraphEntity, dst graphviz.GraphEntity) graphviz.PropertySet {
	properties := make(map[string]string)
	return properties
}

// SubgraphMarker identifies the start of subgraphs for resources.
// Specifically, it starts a new subgraph whenever a new 'Module' type resource
// is encountered.
func (p PreparerProvider) SubgraphMarker(e graphviz.GraphEntity) graphviz.SubgraphMarkerKey {
	switch e.Value.(type) {
	case *module.Preparer:
		return graphviz.SubgraphMarkerStart
	default:
		return graphviz.SubgraphMarkerNOP
	}
}

// NewPreparerProvider is a utility function to return a new PreparerProvider.
func NewPreparerProvider() graphviz.PrintProvider {
	return PreparerProvider{}
}
