//go:build integ
// +build integ

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

package pilot

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/onsi/gomega"

	"istio.io/istio/istioctl/pkg/analyze"
	"istio.io/istio/pkg/config/analysis/diag"
	"istio.io/istio/pkg/config/analysis/msg"
	"istio.io/istio/pkg/test"
	"istio.io/istio/pkg/test/framework"
	"istio.io/istio/pkg/test/framework/components/istioctl"
	"istio.io/istio/pkg/test/framework/components/namespace"
)

const (
	gatewayFile          = "testdata/gateway.yaml"
	jsonGatewayFile      = "testdata/gateway.json"
	destinationRuleFile  = "testdata/destinationrule.yaml"
	virtualServiceFile   = "testdata/virtualservice.yaml"
	invalidFile          = "testdata/invalid.yaml"
	invalidExtensionFile = "testdata/invalid.md"
	dirWithConfig        = "testdata/some-dir/"
	jsonOutput           = "-ojson"
)

var analyzerFoundIssuesError = analyze.AnalyzerFoundIssuesError{}

func TestEmptyCluster(t *testing.T) {
	// nolint: staticcheck
	framework.
		NewTest(t).
		RequiresSingleCluster().
		Run(func(t framework.TestContext) {
			g := gomega.NewWithT(t)

			ns := namespace.NewOrFail(t, t, namespace.Config{
				Prefix: "istioctl-analyze",
				Inject: true,
			})

			istioCtl := istioctl.NewOrFail(t, t, istioctl.Config{})

			// For a clean istio install with injection enabled, expect no validation errors
			output, err := istioctlSafe(t, istioCtl, ns.Name(), true)
			expectNoMessages(t, g, output)
			g.Expect(err).To(gomega.BeNil())
		})
}

func TestFileOnly(t *testing.T) {
	// nolint: staticcheck
	framework.
		NewTest(t).
		RequiresSingleCluster().
		Run(func(t framework.TestContext) {
			g := gomega.NewWithT(t)

			ns := namespace.NewOrFail(t, t, namespace.Config{
				Prefix: "istioctl-analyze",
				Inject: true,
			})

			istioCtl := istioctl.NewOrFail(t, t, istioctl.Config{})

			// Validation error if we have a virtual service with subset not defined.
			output, err := istioctlSafe(t, istioCtl, ns.Name(), false, virtualServiceFile)
			expectMessages(t, g, output, msg.ReferencedResourceNotFound)
			g.Expect(err).To(gomega.BeIdenticalTo(analyzerFoundIssuesError))

			// Error goes away if we define the subset in the destination rule.
			output, err = istioctlSafe(t, istioCtl, ns.Name(), false, destinationRuleFile)
			expectNoMessages(t, g, output)
			g.Expect(err).To(gomega.BeNil())
		})
}

func TestDirectoryWithoutRecursion(t *testing.T) {
	// nolint: staticcheck
	framework.
		NewTest(t).
		RequiresSingleCluster().
		Run(func(t framework.TestContext) {
			g := gomega.NewWithT(t)

			ns := namespace.NewOrFail(t, t, namespace.Config{
				Prefix: "istioctl-analyze",
				Inject: true,
			})

			istioCtl := istioctl.NewOrFail(t, t, istioctl.Config{})

			// Recursive is false, so we should only analyze
			// testdata/some-dir/missing-gateway.yaml and get a
			// SchemaValidationError (if we did recurse, we'd get a
			// UnknownAnnotation as well).
			output, err := istioctlSafe(t, istioCtl, ns.Name(), false, "--recursive=false", dirWithConfig)
			expectMessages(t, g, output, msg.SchemaValidationError)
			g.Expect(err).To(gomega.BeIdenticalTo(analyzerFoundIssuesError))
		})
}

func TestDirectoryWithRecursion(t *testing.T) {
	// nolint: staticcheck
	framework.
		NewTest(t).
		RequiresSingleCluster().
		Run(func(t framework.TestContext) {
			g := gomega.NewWithT(t)

			ns := namespace.NewOrFail(t, t, namespace.Config{
				Prefix: "istioctl-analyze",
				Inject: true,
			})

			istioCtl := istioctl.NewOrFail(t, t, istioctl.Config{})

			// Recursive is true, so we should see one error (SchemaValidationError).
			output, err := istioctlSafe(t, istioCtl, ns.Name(), false, "--recursive=true", dirWithConfig)
			expectMessages(t, g, output, msg.SchemaValidationError)
			g.Expect(err).To(gomega.BeIdenticalTo(analyzerFoundIssuesError))
		})
}

func TestInvalidFileError(t *testing.T) {
	// nolint: staticcheck
	framework.
		NewTest(t).
		RequiresSingleCluster().
		Run(func(t framework.TestContext) {
			g := gomega.NewWithT(t)

			ns := namespace.NewOrFail(t, t, namespace.Config{
				Prefix: "istioctl-analyze",
				Inject: true,
			})

			istioCtl := istioctl.NewOrFail(t, t, istioctl.Config{})

			// Skip the file with invalid extension and produce no errors.
			output, err := istioctlSafe(t, istioCtl, ns.Name(), false, invalidExtensionFile)
			g.Expect(output[0]).To(gomega.ContainSubstring(fmt.Sprintf("Skipping file %v, recognized file extensions are: [.json .yaml .yml]", invalidExtensionFile)))
			g.Expect(err).To(gomega.BeNil())

			// Parse error as the yaml file itself is not valid yaml.
			output, err = istioctlSafe(t, istioCtl, ns.Name(), false, invalidFile)
			g.Expect(strings.Join(output, "\n")).To(gomega.ContainSubstring("Error(s) adding files"))
			g.Expect(strings.Join(output, "\n")).To(gomega.ContainSubstring(fmt.Sprintf("errors parsing content \"%s\"", invalidFile)))

			g.Expect(err).To(gomega.MatchError(analyze.FileParseError{}))

			// Parse error as the yaml file itself is not valid yaml, but ignore.
			output, err = istioctlSafe(t, istioCtl, ns.Name(), false, invalidFile, "--ignore-unknown=true")
			g.Expect(strings.Join(output, "\n")).To(gomega.ContainSubstring("Error(s) adding files"))
			g.Expect(strings.Join(output, "\n")).To(gomega.ContainSubstring(fmt.Sprintf("errors parsing content \"%s\"", invalidFile)))

			g.Expect(err).To(gomega.BeNil())
		})
}

func TestJsonInputFile(t *testing.T) {
	// nolint: staticcheck
	framework.
		NewTest(t).
		RequiresSingleCluster().
		Run(func(t framework.TestContext) {
			g := gomega.NewWithT(t)

			ns := namespace.NewOrFail(t, t, namespace.Config{
				Prefix: "istioctl-analyze",
				Inject: true,
			})

			istioCtl := istioctl.NewOrFail(t, t, istioctl.Config{})

			// Validation error if we have a gateway with invalid selector.
			applyFileOrFail(t, ns.Name(), jsonGatewayFile)
			output, err := istioctlSafe(t, istioCtl, ns.Name(), true)
			expectMessages(t, g, output, msg.ReferencedResourceNotFound)
			g.Expect(err).To(gomega.BeIdenticalTo(analyzerFoundIssuesError))
		})
}

func TestJsonOutput(t *testing.T) {
	// nolint: staticcheck
	framework.
		NewTest(t).
		RequiresSingleCluster().
		Run(func(t framework.TestContext) {
			g := gomega.NewWithT(t)

			ns := namespace.NewOrFail(t, t, namespace.Config{
				Prefix: "istioctl-analyze",
				Inject: true,
			})

			istioCtl := istioctl.NewOrFail(t, t, istioctl.Config{})

			t.NewSubTest("no other output except analysis json output").Run(func(t framework.TestContext) {
				applyFileOrFail(t, ns.Name(), jsonGatewayFile)
				stdout, _, err := istioctlWithStderr(t, istioCtl, ns.Name(), true, jsonOutput)
				expectJSONMessages(t, g, stdout, msg.ReferencedResourceNotFound)
				g.Expect(err).To(gomega.BeNil())
			})

			t.NewSubTest("invalid file does not output error in stdout").Run(func(t framework.TestContext) {
				stdout, _, err := istioctlWithStderr(t, istioCtl, ns.Name(), false, invalidExtensionFile, jsonOutput)
				expectJSONMessages(t, g, stdout)
				g.Expect(err).To(gomega.BeNil())
			})
		})
}

func TestKubeOnly(t *testing.T) {
	// nolint: staticcheck
	framework.
		NewTest(t).
		RequiresSingleCluster().
		Run(func(t framework.TestContext) {
			g := gomega.NewWithT(t)

			ns := namespace.NewOrFail(t, t, namespace.Config{
				Prefix: "istioctl-analyze",
				Inject: true,
			})

			applyFileOrFail(t, ns.Name(), gatewayFile)

			istioCtl := istioctl.NewOrFail(t, t, istioctl.Config{})

			// Validation error if we have a gateway with invalid selector.
			output, err := istioctlSafe(t, istioCtl, ns.Name(), true)
			expectMessages(t, g, output, msg.ReferencedResourceNotFound)
			g.Expect(err).To(gomega.BeIdenticalTo(analyzerFoundIssuesError))
		})
}

func TestFileAndKubeCombined(t *testing.T) {
	// nolint: staticcheck
	framework.
		NewTest(t).
		RequiresSingleCluster().
		Run(func(t framework.TestContext) {
			g := gomega.NewWithT(t)

			ns := namespace.NewOrFail(t, t, namespace.Config{
				Prefix: "istioctl-analyze",
				Inject: true,
			})

			applyFileOrFail(t, ns.Name(), virtualServiceFile)

			istioCtl := istioctl.NewOrFail(t, t, istioctl.Config{})

			// Simulating applying the destination rule that defines the subset, we should
			// fix the error and thus see no message
			output, err := istioctlSafe(t, istioCtl, ns.Name(), true, destinationRuleFile)
			expectNoMessages(t, g, output)
			g.Expect(err).To(gomega.BeNil())
		})
}

func TestAllNamespaces(t *testing.T) {
	// nolint: staticcheck
	framework.
		NewTest(t).
		RequiresSingleCluster().
		Run(func(t framework.TestContext) {
			g := gomega.NewWithT(t)

			ns1 := namespace.NewOrFail(t, t, namespace.Config{
				Prefix: "istioctl-analyze-1",
				Inject: true,
			})
			ns2 := namespace.NewOrFail(t, t, namespace.Config{
				Prefix: "istioctl-analyze-2",
				Inject: true,
			})

			applyFileOrFail(t, ns1.Name(), gatewayFile)
			applyFileOrFail(t, ns2.Name(), gatewayFile)

			istioCtl := istioctl.NewOrFail(t, t, istioctl.Config{})

			// If we look at one namespace, we should successfully run and see one message (and not anything from any other namespace)
			output, _ := istioctlSafe(t, istioCtl, ns1.Name(), true)
			expectMessages(t, g, output, msg.ReferencedResourceNotFound, msg.ConflictingGateways)

			// If we use --all-namespaces, we should successfully run and see a message from each namespace
			output, _ = istioctlSafe(t, istioCtl, "", true, "--all-namespaces")
			// Since this test runs in a cluster with lots of other namespaces we don't actually care about, only look for ns1 and ns2
			foundCount := 0
			for _, line := range output {
				if strings.Contains(line, ns1.Name()) {
					if strings.Contains(line, msg.ReferencedResourceNotFound.Code()) {
						g.Expect(line).To(gomega.ContainSubstring(msg.ReferencedResourceNotFound.Code()))
						foundCount++
					}
					// There are 2 conflictings can be detected, A to B and B to A
					if strings.Contains(line, msg.ConflictingGateways.Code()) {
						g.Expect(line).To(gomega.ContainSubstring(msg.ConflictingGateways.Code()))
						foundCount++
					}
				}
				if strings.Contains(line, ns2.Name()) {
					if strings.Contains(line, msg.ReferencedResourceNotFound.Code()) {
						g.Expect(line).To(gomega.ContainSubstring(msg.ReferencedResourceNotFound.Code()))
						foundCount++
					}
					// There are 2 conflictings can be detected, B to A and A to B
					if strings.Contains(line, msg.ConflictingGateways.Code()) {
						g.Expect(line).To(gomega.ContainSubstring(msg.ConflictingGateways.Code()))
						foundCount++
					}
				}
			}
			g.Expect(foundCount).To(gomega.Equal(6))
		})
}

func TestTimeout(t *testing.T) {
	t.Skip("https://github.com/istio/istio/issues/25893")
	// nolint: staticcheck
	framework.
		NewTest(t).
		RequiresSingleCluster().
		Run(func(t framework.TestContext) {
			g := gomega.NewWithT(t)

			ns := namespace.NewOrFail(t, t, namespace.Config{
				Prefix: "istioctl-analyze",
				Inject: true,
			})

			istioCtl := istioctl.NewOrFail(t, t, istioctl.Config{})

			// We should time out immediately.
			_, err := istioctlSafe(t, istioCtl, ns.Name(), true, "--timeout=0s")
			g.Expect(err.Error()).To(gomega.ContainSubstring("timed out"))
		})
}

// Verify the error line number in the message is correct
func TestErrorLine(t *testing.T) {
	// nolint: staticcheck
	framework.
		NewTest(t).
		RequiresSingleCluster().
		Features("usability.observability.analysis.line-numbers").
		Run(func(t framework.TestContext) {
			g := gomega.NewWithT(t)

			ns := namespace.NewOrFail(t, t, namespace.Config{
				Prefix: "istioctl-analyze",
				Inject: true,
			})

			istioCtl := istioctl.NewOrFail(t, t, istioctl.Config{})

			// Validation error if we have a gateway with invalid selector.
			output, err := istioctlSafe(t, istioCtl, ns.Name(), true, gatewayFile, virtualServiceFile)

			g.Expect(strings.Join(output, "\n")).To(gomega.ContainSubstring("testdata/gateway.yaml:9"))
			g.Expect(strings.Join(output, "\n")).To(gomega.ContainSubstring("testdata/virtualservice.yaml:11"))
			g.Expect(err).To(gomega.BeIdenticalTo(analyzerFoundIssuesError))
		})
}

// Verify the output contains messages of the expected type, in order, followed by boilerplate lines
func expectMessages(t test.Failer, g *gomega.GomegaWithT, outputLines []string, expected ...*diag.MessageType) {
	t.Helper()

	// The boilerplate lines that appear if any issues are found
	boilerplateLines := strings.Split(analyzerFoundIssuesError.Error(), "\n")

	g.Expect(outputLines).To(gomega.HaveLen(len(expected) + len(boilerplateLines)))

	for i, line := range outputLines {
		if i < len(expected) {
			g.Expect(line).To(gomega.ContainSubstring(expected[i].Code()))
		} else {
			g.Expect(line).To(gomega.ContainSubstring(boilerplateLines[i-len(expected)]))
		}
	}
}

func expectNoMessages(t test.Failer, g *gomega.GomegaWithT, output []string) {
	t.Helper()
	g.Expect(output).To(gomega.HaveLen(1))
	g.Expect(output[0]).To(gomega.ContainSubstring("No validation issues found when analyzing"))
}

func expectJSONMessages(t test.Failer, g *gomega.GomegaWithT, output string, expected ...*diag.MessageType) {
	t.Helper()

	var j []map[string]any
	if err := json.Unmarshal([]byte(output), &j); err != nil {
		t.Fatal(err, output)
	}

	g.Expect(j).To(gomega.HaveLen(len(expected)))

	for i, m := range j {
		g.Expect(m["level"]).To(gomega.Equal(expected[i].Level().String()))
		g.Expect(m["code"]).To(gomega.Equal(expected[i].Code()))
	}
}

// istioctlSafe calls istioctl analyze with certain flags set. Stdout and Stderr are merged
func istioctlSafe(t test.Failer, i istioctl.Instance, ns string, useKube bool, extraArgs ...string) ([]string, error) {
	output, stderr, err := istioctlWithStderr(t, i, ns, useKube, extraArgs...)
	return strings.Split(strings.TrimSpace(output+stderr), "\n"), err
}

func istioctlWithStderr(t test.Failer, i istioctl.Instance, ns string, useKube bool, extraArgs ...string) (string, string, error) {
	t.Helper()

	args := []string{"analyze"}
	if ns != "" {
		args = append(args, "--namespace", ns)
	}
	// Suppress some cluster-wide checks. This ensures we do not fail tests when running on clusters that trigger
	// analyzers we didn't intended to test.
	args = append(args, fmt.Sprintf("--use-kube=%t", useKube), "--suppress=IST0139=*", "--suppress=IST0002=CustomResourceDefinition *")
	args = append(args, extraArgs...)

	return i.Invoke(args)
}

// applyFileOrFail applys the given yaml file and deletes it during context cleanup
func applyFileOrFail(t framework.TestContext, ns, filename string) {
	t.Helper()
	if err := t.Clusters().Default().ApplyYAMLFiles(ns, filename); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = t.Clusters().Default().DeleteYAMLFiles(ns, filename)
	})
}
