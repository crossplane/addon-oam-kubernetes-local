package webhooks

import (
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestWebhooks(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Webhooks Suite")
}

var _ = BeforeSuite(func(done Done) {
	By("Bootstrapping test environment")
	logf.SetLogger(zap.New(zap.UseDevMode(true), zap.WriteTo(GinkgoWriter)))
	close(done)
}, 300)
