// Copyright Istio Authors
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

package util

import (
	"testing"

	"github.com/onsi/gomega"
)

func TestIsExportToAllNamespaces(t *testing.T) {
	g := gomega.NewWithT(t)

	// Empty array
	g.Expect(IsExportToAllNamespaces(nil)).To(gomega.Equal(true))

	// Array with "*"
	g.Expect(IsExportToAllNamespaces([]string{"*"})).To(gomega.Equal(true))

	// Array with "."
	g.Expect(IsExportToAllNamespaces([]string{"."})).To(gomega.Equal(false))

	// Array with "." & "*"
	g.Expect(IsExportToAllNamespaces([]string{".", "*"})).To(gomega.Equal(true))

	// Array with "bogus"
	g.Expect(IsExportToAllNamespaces([]string{"bogus"})).To(gomega.Equal(false))
}
