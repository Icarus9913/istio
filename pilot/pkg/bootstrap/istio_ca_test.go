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

package bootstrap

import (
	"os"
	"path"
	"testing"

	"github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"istio.io/istio/pkg/kube"
	"istio.io/istio/pkg/kube/kclient/clienttest"
	"istio.io/istio/pkg/test"
	"istio.io/istio/pkg/test/env"
	"istio.io/istio/security/pkg/pki/ca"
)

const testNamespace = "istio-system"

func TestRemoteCerts(t *testing.T) {
	g := gomega.NewWithT(t)

	dir := t.TempDir()

	s := Server{
		kubeClient: kube.NewFakeClient(),
	}
	s.kubeClient.RunAndWait(test.NewStop(t))
	caOpts := &caOptions{
		Namespace: testNamespace,
	}

	// Should do nothing because cacerts doesn't exist.
	err := s.loadCACerts(caOpts, dir)
	g.Expect(err).Should(gomega.BeNil())

	_, err = os.Stat(path.Join(dir, "root-cert.pem"))
	g.Expect(os.IsNotExist(err)).Should(gomega.Equal(true))

	// Should load remote cacerts successfully.
	createCASecret(t, s.kubeClient)

	err = s.loadCACerts(caOpts, dir)
	g.Expect(err).Should(gomega.BeNil())

	expectedRoot, err := readSampleCertFromFile("root-cert.pem")
	g.Expect(err).Should(gomega.BeNil())

	g.Expect(os.ReadFile(path.Join(dir, "root-cert.pem"))).Should(gomega.Equal(expectedRoot))

	// Should do nothing because certs already exist locally.
	err = s.loadCACerts(caOpts, dir)
	g.Expect(err).Should(gomega.BeNil())
}

func TestRemoteTLSCerts(t *testing.T) {
	g := gomega.NewWithT(t)

	dir := t.TempDir()

	s := Server{
		kubeClient: kube.NewFakeClient(),
	}
	s.kubeClient.RunAndWait(test.NewStop(t))
	caOpts := &caOptions{
		Namespace: testNamespace,
	}

	// Should do nothing because cacerts doesn't exist.
	err := s.loadCACerts(caOpts, dir)
	g.Expect(err).Should(gomega.BeNil())

	_, err = os.Stat(path.Join(dir, "ca.crt"))
	g.Expect(os.IsNotExist(err)).Should(gomega.Equal(true))

	// Should load remote cacerts successfully.
	createCATLSSecret(t, s.kubeClient)

	err = s.loadCACerts(caOpts, dir)
	g.Expect(err).Should(gomega.BeNil())

	expectedRoot, err := readSampleCertFromFile("root-cert.pem")
	g.Expect(err).Should(gomega.BeNil())

	g.Expect(os.ReadFile(path.Join(dir, "ca.crt"))).Should(gomega.Equal(expectedRoot))

	// Should do nothing because certs already exist locally.
	err = s.loadCACerts(caOpts, dir)
	g.Expect(err).Should(gomega.BeNil())
}

func createCATLSSecret(t test.Failer, client kube.Client) {
	var caCert, caKey, rootCert []byte
	var err error
	if caCert, err = readSampleCertFromFile("ca-cert.pem"); err != nil {
		t.Fatal(err)
	}
	if caKey, err = readSampleCertFromFile("ca-key.pem"); err != nil {
		t.Fatal(err)
	}
	if rootCert, err = readSampleCertFromFile("root-cert.pem"); err != nil {
		t.Fatal(err)
	}

	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      "cacerts",
		},
		Type: v1.SecretTypeTLS,
		Data: map[string][]byte{
			"tls.crt": caCert,
			"tls.key": caKey,
			"ca.crt":  rootCert,
		},
	}
	clienttest.NewWriter[*v1.Secret](t, client).Create(secret)
}

func createCASecret(t test.Failer, client kube.Client) {
	var caCert, caKey, certChain, rootCert []byte
	var err error
	if caCert, err = readSampleCertFromFile("ca-cert.pem"); err != nil {
		t.Fatal(err)
	}
	if caKey, err = readSampleCertFromFile("ca-key.pem"); err != nil {
		t.Fatal(err)
	}
	if certChain, err = readSampleCertFromFile("cert-chain.pem"); err != nil {
		t.Fatal(err)
	}
	if rootCert, err = readSampleCertFromFile("root-cert.pem"); err != nil {
		t.Fatal(err)
	}

	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      "cacerts",
		},
		Data: map[string][]byte{
			ca.CACertFile:       caCert,
			ca.CAPrivateKeyFile: caKey,
			ca.CertChainFile:    certChain,
			ca.RootCertFile:     rootCert,
		},
	}

	clienttest.NewWriter[*v1.Secret](t, client).Create(secret)
}

func readSampleCertFromFile(f string) ([]byte, error) {
	return os.ReadFile(path.Join(env.IstioSrc, "samples/certs", f))
}
