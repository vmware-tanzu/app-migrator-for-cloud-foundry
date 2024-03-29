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

// Code generated by counterfeiter. DO NOT EDIT.
package fakes

import (
	"sync"

	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/context"
)

type FakeAppCommandRunner struct {
	RunStub        func(*context.Context, string, string) error
	runMutex       sync.RWMutex
	runArgsForCall []struct {
		arg1 *context.Context
		arg2 string
		arg3 string
	}
	runReturns struct {
		result1 error
	}
	runReturnsOnCall map[int]struct {
		result1 error
	}
	SetAppNameStub        func(string)
	setAppNameMutex       sync.RWMutex
	setAppNameArgsForCall []struct {
		arg1 string
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeAppCommandRunner) Run(arg1 *context.Context, arg2 string, arg3 string) error {
	fake.runMutex.Lock()
	ret, specificReturn := fake.runReturnsOnCall[len(fake.runArgsForCall)]
	fake.runArgsForCall = append(fake.runArgsForCall, struct {
		arg1 *context.Context
		arg2 string
		arg3 string
	}{arg1, arg2, arg3})
	stub := fake.RunStub
	fakeReturns := fake.runReturns
	fake.recordInvocation("Run", []interface{}{arg1, arg2, arg3})
	fake.runMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2, arg3)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeAppCommandRunner) RunCallCount() int {
	fake.runMutex.RLock()
	defer fake.runMutex.RUnlock()
	return len(fake.runArgsForCall)
}

func (fake *FakeAppCommandRunner) RunCalls(stub func(*context.Context, string, string) error) {
	fake.runMutex.Lock()
	defer fake.runMutex.Unlock()
	fake.RunStub = stub
}

func (fake *FakeAppCommandRunner) RunArgsForCall(i int) (*context.Context, string, string) {
	fake.runMutex.RLock()
	defer fake.runMutex.RUnlock()
	argsForCall := fake.runArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2, argsForCall.arg3
}

func (fake *FakeAppCommandRunner) RunReturns(result1 error) {
	fake.runMutex.Lock()
	defer fake.runMutex.Unlock()
	fake.RunStub = nil
	fake.runReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeAppCommandRunner) RunReturnsOnCall(i int, result1 error) {
	fake.runMutex.Lock()
	defer fake.runMutex.Unlock()
	fake.RunStub = nil
	if fake.runReturnsOnCall == nil {
		fake.runReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.runReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeAppCommandRunner) SetAppName(arg1 string) {
	fake.setAppNameMutex.Lock()
	fake.setAppNameArgsForCall = append(fake.setAppNameArgsForCall, struct {
		arg1 string
	}{arg1})
	stub := fake.SetAppNameStub
	fake.recordInvocation("SetAppName", []interface{}{arg1})
	fake.setAppNameMutex.Unlock()
	if stub != nil {
		fake.SetAppNameStub(arg1)
	}
}

func (fake *FakeAppCommandRunner) SetAppNameCallCount() int {
	fake.setAppNameMutex.RLock()
	defer fake.setAppNameMutex.RUnlock()
	return len(fake.setAppNameArgsForCall)
}

func (fake *FakeAppCommandRunner) SetAppNameCalls(stub func(string)) {
	fake.setAppNameMutex.Lock()
	defer fake.setAppNameMutex.Unlock()
	fake.SetAppNameStub = stub
}

func (fake *FakeAppCommandRunner) SetAppNameArgsForCall(i int) string {
	fake.setAppNameMutex.RLock()
	defer fake.setAppNameMutex.RUnlock()
	argsForCall := fake.setAppNameArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeAppCommandRunner) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.runMutex.RLock()
	defer fake.runMutex.RUnlock()
	fake.setAppNameMutex.RLock()
	defer fake.setAppNameMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeAppCommandRunner) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}

var _ context.AppCommandRunner = new(FakeAppCommandRunner)
