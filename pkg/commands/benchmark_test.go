/* 
 *  Copyright 2022 VMware, Inc.
 *  
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *  http://www.apache.org/licenses/LICENSE-2.0
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

package commands_test

import (
	"strings"
	"testing"
	"time"
)

var compareResult int
var afterResult bool

func BenchmarkFormatAndCompare(b *testing.B) {
	now := time.Now()
	var r int
	for i := 0; i < b.N; i++ {
		t := now.Add(time.Duration(i) * time.Second)
		x := t.Format(time.RFC3339)

		s := t.Format(time.RFC3339)
		r = strings.Compare(s, x)
	}

	compareResult = r
}

func BenchmarkParseAndAfter(b *testing.B) {
	now := time.Now()

	var a bool
	for i := 0; i < b.N; i++ {
		t := now.Add(time.Duration(i) * time.Second)
		s := t.Format(time.RFC3339)

		x, _ := time.Parse(time.RFC3339, s)
		a = t.After(x)
	}

	afterResult = a
}
