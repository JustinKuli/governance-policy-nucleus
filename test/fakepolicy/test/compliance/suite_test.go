// Copyright Contributors to the Open Cluster Management project

package compliance

import (
	"context"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"open-cluster-management.io/governance-policy-nucleus/pkg/testutils"
	"open-cluster-management.io/governance-policy-nucleus/test/fakepolicy"
	fakev1beta1 "open-cluster-management.io/governance-policy-nucleus/test/fakepolicy/api/v1beta1"
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	cfg     *rest.Config
	testEnv *envtest.Environment
	ctx     context.Context
	cancel  context.CancelFunc
	tk      testutils.Toolkit
)

//nolint:paralleltest // scaffolded this way by ginkgo
func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Compliance Details Suite")
}

var _ = BeforeSuite(func() {
	format.TruncatedDiff = false
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.TODO())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	var err error
	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = fakev1beta1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	tk, err = testutils.NewToolkitFromRest(cfg, "")
	Expect(err).NotTo(HaveOccurred())
	tk = tk.WithCtx(ctx)

	go func() {
		defer GinkgoRecover()
		Expect(fakepolicy.Run(ctx, cfg)).To(Succeed())
	}()
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	cancel()
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
