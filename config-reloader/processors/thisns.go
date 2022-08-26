// Copyright © 2018 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: BSD-2-Clause

package processors

import (
	"fmt"
	"strings"

	"github.com/vmware/kube-fluentd-operator/config-reloader/fluentd"
)

const (
	macroThisns = "$thisns"
)

type expandThisnsMacroState struct {
	BaseProcessorState
}

func (p *expandThisnsMacroState) Process(input fluentd.Fragment) (fluentd.Fragment, error) {
	f := func(d *fluentd.Directive, ctx *ProcessorContext) error {
		namespace := ctx.Namespace

		if d.Name != "match" &&
			d.Name != "filter" {
			return nil
		}

		goodPrefix := fmt.Sprintf("kube.%s", namespace)

		if d.Tag == "**" || d.Tag == macroThisns {
			d.Tag = goodPrefix + ".**"
			ctx.GenerationContext.augmentTag(d)
			return nil
		}

		if strings.HasPrefix(d.Tag, macroThisns) {
			// handle the unusual case of $thisns.**
			d.Tag = goodPrefix + d.Tag[len(macroThisns):]
			ctx.GenerationContext.augmentTag(d)
			return nil
		}

		if strings.HasPrefix(d.Tag, macroLabels) || strings.HasPrefix(d.Tag, macroUniqueTag) {
			// Let other processors handle this
			return nil
		}

		s := strings.ReplaceAll(d.Tag, macroThisns, goodPrefix)

		if !strings.HasPrefix(s, goodPrefix+".") {
			return fmt.Errorf("bad tag for <%s>: %s. Tag must start with **, $thisns or %s", d.Name, d.Tag, namespace)
		}

		return nil
	}

	// we check top level directives here before going into recursion, since
	// inside recursion we cannot determine whether we are at top level
	if p.Context.Strict {
		for _, d := range input {
			if d.Name != "match" && d.Name != "filter" {
				return nil, fmt.Errorf(
					"strict mode only allows 'match' and 'filter' tags, not '%s'", d.Name)
			}
		}
	}

	err := applyRecursivelyInPlace(input, p.Context, f)
	if err != nil {
		return nil, err
	}

	return input, nil
}
