// Copyright © 2018 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: BSD-2-Clause

package util

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os/exec"
	"sort"
	"strings"
	"time"
	"unicode"
)

const (
	maskFile = 0664
)

func Trim(s string) string {
	return strings.TrimSpace(s)
}

func MakeFluentdSafeName(s string) string {
	buf := &bytes.Buffer{}
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' && r != '_' {
			buf.WriteRune('-')
		} else {
			buf.WriteRune(r)
		}
	}

	return buf.String()
}

func ToRubyMapLiteral(labels map[string]string) string {
	if len(labels) == 0 {
		return "{}"
	}

	buf := &bytes.Buffer{}
	buf.WriteString("{")
	for _, k := range SortedKeys(labels) {
		fmt.Fprintf(buf, "'%s'=>'%s',", k, labels[k])
	}
	buf.Truncate(buf.Len() - 1)
	buf.WriteString("}")

	return buf.String()
}

func Hash(owner string, value string) string {
	h := sha256.New()

	h.Write([]byte(owner))
	h.Write([]byte(":"))
	h.Write([]byte(value))

	b := h.Sum(nil)
	return hex.EncodeToString(b[0:20])
}

func SortedKeys(m map[string]string) []string {
	keys := make([]string, len(m))
	i := 0

	for k := range m {
		keys[i] = k
		i++
	}
	sort.Strings(keys)

	return keys
}

// ExecAndGetOutput exec and returns output of the command if timeout then kills the process and returns error
func ExecAndGetOutput(cmd string, timeout time.Duration, args ...string) (string, error) {
	c := exec.Command(cmd, args...)
	var b bytes.Buffer
	c.Stdout = &b
	c.Stderr = &b
	var err error
	if err = c.Start(); err != nil {
		out := b.Bytes()
		return string(out), err
	}

	// Wait for the process to finish or kill it after a timeout (whichever happens first):
	done := make(chan error, 1)
	go func() {
		done <- c.Wait()
	}()

	select {
	case <-time.After(timeout):
		if err = c.Process.Kill(); err != nil {
			err = fmt.Errorf("process killed as timeout reached after %s, but kill failed with err: %s", timeout, err.Error())
		} else {
			err = fmt.Errorf("process killed as timeout reached after %s", timeout)
		}
	case err = <-done:
	}
	out := b.Bytes()

	return string(out), err
}

func WriteStringToFile(filename string, data string) error {
	return ioutil.WriteFile(filename, []byte(data), maskFile)
}

func TrimTrailingComment(line string) string {
	i := strings.IndexByte(line, '#')
	if i > 0 {
		line = Trim(line[0:i])
	} else {
		line = Trim(line)
	}

	return line
}

func MakeStructureHash(v interface{}) (uint64, error) {
	hashV, err := hashstructure.Hash(v, hashstructure.FormatV2, nil)
	if err != nil {
		return hashV, err
	}

	return hashV, nil
}

func AreStructureHashEqual(v interface{}, f interface{}) bool {
	hashV, _ := hashstructure.Hash(v, hashstructure.FormatV2, nil)
	hashF, _ := hashstructure.Hash(f, hashstructure.FormatV2, nil)

	if hashV != 0 && hashF != 0 {
		return hashV == hashF
	}

	return false
}

type AllowList struct {
	allowed map[string]bool
}

func NewAllowList(l string) *AllowList {
	ret := &AllowList{allowed: make(map[string]bool)}
	for _, e := range strings.Split(l, ",") {
		ret.Allow(e)
	}
	return ret
}

func (a *AllowList) Allow(k string) {
	if a != nil && len(k) > 0 {
		a.allowed[k] = true
	}
}

func (a *AllowList) Deny(k string) {
	if a != nil {
		delete(a.allowed, k)
	}
}

func (a *AllowList) Ok(k string) bool {
	if a == nil || len(a.allowed) == 0 {
		return true
	}
	return a.allowed[k]
}
