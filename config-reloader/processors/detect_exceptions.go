// Copyright © 2018 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: BSD-2-Clause

package processors

import (
	"fmt"
	"strings"

	"github.com/vmware/kube-fluentd-operator/config-reloader/fluentd"
	"github.com/vmware/kube-fluentd-operator/config-reloader/util"
)

const (
	keyDetExc = "detexc"
)

type detectExceptionsState struct {
	BaseProcessorState
}

func (state *detectExceptionsState) Process(input fluentd.Fragment) (fluentd.Fragment, error) {
	needsRewrite := false
	rewrite := func(dir *fluentd.Directive, parent *fluentd.Fragment) *fluentd.Directive {
		if dir.Name != "filter" || dir.Type() != "detect_exceptions" {
			c := dir.Clone()
			*parent = append(*parent, c)
			return c
		}

		needsRewrite = true
		tagPrefix := makeTagPrefix(dir.Tag)

		rule := &fluentd.Directive{
			Name:   "rule",
			Params: fluentd.Params{},
		}
		rule.SetParam("key", "_dummy")
		rule.SetParam("pattern", "/ZZ/")
		rule.SetParam("invert", "true")
		rule.SetParam("tag", fmt.Sprintf("%s.%s.${tag}", tagPrefix, keyDetExc))

		rewriteTag := &fluentd.Directive{
			Name:   "match",
			Tag:    dir.Tag,
			Params: fluentd.ParamsFromKV("@type", "rewrite_tag_filter"),
			Nested: fluentd.Fragment{rule},
		}

		detectExceptions := &fluentd.Directive{
			Name:   "match",
			Tag:    fmt.Sprintf("%s.%s.%s", tagPrefix, keyDetExc, dir.Tag),
			Params: fluentd.ParamsFromKV("@type", "detect_exceptions"),
		}
		detectExceptions.SetParam("stream", "container_info")
		detectExceptions.SetParam("remove_tag_prefix", tagPrefix)

		// copy all relevant params from the original <filter> directive
		copyParam("languages", dir, detectExceptions)
		copyParam("multiline_flush_interval", dir, detectExceptions)
		copyParam("max_lines", dir, detectExceptions)
		copyParam("max_bytes", dir, detectExceptions)
		copyParam("message", dir, detectExceptions)

		*parent = append(*parent, rewriteTag, detectExceptions)

		return nil
	}

	res := transform(input, rewrite)

	if needsRewrite {
		augmentTag := func(dir *fluentd.Directive, ctx *ProcessorContext) error {
			// only process the original directives

			pfx := fmt.Sprintf("kube.%s.", ctx.Namepsace)
			if strings.HasPrefix(dir.Tag, pfx) &&
				dir.Type() != "rewrite_tag_filter" {
				tag := dir.Tag
				dir.Tag = fmt.Sprintf("%s %s.%s", tag, keyDetExc, tag)
			}

			return nil
		}

		applyRecursivelyInPlace(res, state.Context, augmentTag)
	}

	return res, nil
}

func makeTagPrefix(selector string) string {
	return util.Hash(keyDetExc, selector)
}

func copyParam(name string, src, dest *fluentd.Directive) {
	val := src.Param(name)
	if val != "" {
		dest.SetParam(name, val)
	}
}
