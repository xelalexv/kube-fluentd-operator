// Copyright © 2018 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: BSD-2-Clause

package processors

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vmware/kube-fluentd-operator/config-reloader/fluentd"
)

func TestRewrite(t *testing.T) {
	ctx := &ProcessorContext{
		Namepsace: "monitoring",
		GenerationContext: &GenerationContext{
			ReferencedBridges: map[string]bool{},
		},
	}

	s := `
<filter $labels(app=jpetstore)>
	@type detect_exceptions
	languages java, python
</filter>

<filter $labels(server=apache)>
	@type parse
	format apache2
</filter>

<match **>
  @type null
</match>
`

	fragment, err := fluentd.ParseString(s)
	assert.Nil(t, err)

	detExc := &detectExceptionsState{}
	labelProc := &expandLabelsMacroState{}
	expandThis := &expandThisnsMacroState{}

	_, err = Prepare(fragment, ctx, expandThis, labelProc, detExc)
	assert.Nil(t, err)

	processed, err := Process(fragment, ctx, expandThis, labelProc, detExc)
	assert.Nil(t, err)
	assert.Equal(t, 7, len(processed))

	fmt.Printf("Processed:\n%s\n", processed)
}

func TestBuild(t *testing.T) {
	var s = `
<match **>
    @type logzio
	<buffer>
		@type file
		path /etc/passwd
		<nested>
		</nested>
  </buffer>
</match>

<match **>
  @type logzio
	<buffer>
		@type file
		path /etc/passwd
  </buffer>
</match>
`

	fragment, err := fluentd.ParseString(s)
	assert.Nil(t, err)

	clone := transform(fragment, copy)

	assert.Equal(t, fragment.String(), clone.String())
}

func TestExtractSelector(t *testing.T) {
	assert.Equal(t, "xxx", extractSelector("xxx"))
	assert.Equal(t, "xxx", extractSelector("xxx _proc.xxx"))
	assert.Equal(t, "xxx", extractSelector("xxx what ever man"))
}
