// Copyright 2021 Matrix Origin
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package extend

import (
	"github.com/matrixorigin/matrixone/pkg/container/batch"
	"github.com/matrixorigin/matrixone/pkg/container/types"
	"github.com/matrixorigin/matrixone/pkg/container/vector"
	"github.com/matrixorigin/matrixone/pkg/vm/process"
)

type Extend interface {
	Eq(Extend) bool
	String() string
	IsLogical() bool
	IsConstant() bool
	ReturnType() types.T
	Attributes() []string
	ExtendAttributes() []*Attribute
	Eval(*batch.Batch, *process.Process) (*vector.Vector, types.T, error)
}

type UnaryExtend struct {
	Op int
	E  Extend
}

type BinaryExtend struct {
	Op          int
	Left, Right Extend
}

type MultiExtend struct {
	Op   int
	Args []Extend
}

type ParenExtend struct {
	E Extend
}

type FuncExtend struct {
	Name string
	Args []Extend
}

type StarExtend struct {
}

type ValueExtend struct {
	V *vector.Vector
}

type Attribute struct {
	Name string  `json:"name"`
	Type types.T `json:"type"`
}
