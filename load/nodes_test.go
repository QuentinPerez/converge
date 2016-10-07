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

package load_test

import (
	"context"
	"sort"
	"testing"

	"github.com/asteris-llc/converge/graph"
	"github.com/asteris-llc/converge/helpers/logging"
	"github.com/asteris-llc/converge/load"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNodesBasic(t *testing.T) {
	defer logging.HideLogs(t)()

	_, err := load.Nodes(context.Background(), "../samples/basic.hcl", false)
	assert.NoError(t, err)
}

func TestNodesSourceFile(t *testing.T) {
	defer logging.HideLogs(t)()

	g, err := load.Nodes(context.Background(), "../samples/sourceFile.hcl", false)
	require.NoError(t, err)

	assert.NotNil(t, g.Get("root/param.message"))
	assert.NotNil(t, g.Get("root/module.basic"))
	assert.NotNil(t, g.Get("root/module.basic/param.message"))
	assert.NotNil(t, g.Get("root/module.basic/param.filename"))
	assert.NotNil(t, g.Get("root/module.basic/task.render"))

	basicDeps := graph.Targets(g.DownEdges("root/module.basic"))
	sort.Strings(basicDeps)

	assert.Equal(
		t,
		[]string{
			"root/module.basic/param.filename",
			"root/module.basic/param.message",
			"root/module.basic/task.render",
		},
		basicDeps,
	)
}

func TestNodeWithConditionals(t *testing.T) {
	defer logging.HideLogs(t)()
	g, err := load.Nodes(context.Background(), "../samples/conditionalLanguages.hcl", false)
	require.NoError(t, err)
	assert.NotNil(t, g.Get("root/param.lang"))
	assert.NotNil(t, g.Get("root/macro.switch.test-switch"))
	assert.NotNil(t, g.Get("root/macro.switch.test-switch/macro.case.spanish"))
	assert.NotNil(t, g.Get("root/macro.switch.test-switch/macro.case.french"))
	assert.NotNil(t, g.Get("root/macro.switch.test-switch/macro.case.japanese"))
	assert.NotNil(t, g.Get("root/macro.switch.test-switch/macro.case.default"))
	assert.NotNil(t, g.Get("root/macro.switch.test-switch/macro.case.spanish/file.content.foo-file"))
	assert.NotNil(t, g.Get("root/macro.switch.test-switch/macro.case.french/file.content.foo-file"))
	assert.NotNil(t, g.Get("root/macro.switch.test-switch/macro.case.japanese/file.content.foo-file"))
	assert.NotNil(t, g.Get("root/macro.switch.test-switch/macro.case.default/file.content.foo-file"))
}
