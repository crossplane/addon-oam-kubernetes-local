/*

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers_test

import (
	"testing"

	"github.com/crossplane/oam-kubernetes-runtime/apis/core"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	// +kubebuilder:scaffold:imports
)

var k8sClient client.Client
var scheme = runtime.NewScheme()

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"OAM Core Resource Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func(done Done) {
	By("Bootstrapping test environment")
	logf.SetLogger(zap.New(zap.UseDevMode(true), zap.WriteTo(GinkgoWriter)))
	err := clientgoscheme.AddToScheme(scheme)
	Expect(err).Should(BeNil())
	err = core.AddToScheme(scheme)
	Expect(err).Should(BeNil())
	By("Setting up kubernetes client")
	k8sClient, err = client.New(config.GetConfigOrDie(), client.Options{Scheme: scheme})
	if err != nil {
		logf.Log.Error(err, "failed to create k8sClient")
		Fail("setup failed")
	}
	By("Finished setting up test environment")
	close(done)
}, 300)

var _ = AfterSuite(func() {
	By("Tearing down the test environment")
})
