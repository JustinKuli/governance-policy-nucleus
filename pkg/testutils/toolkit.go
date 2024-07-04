// Copyright Contributors to the Open Cluster Management project

package testutils

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	gomegaTypes "github.com/onsi/gomega/types"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	ErrNoKubeconfg = errors.New("no known kubeconfig: can not run kubectl")
	ErrKubectl     = errors.New("kubectl exited with error")
)

type Toolkit struct {
	client.Client
	Ctx                 context.Context //nolint:containedctx // this is for convenience
	RestConfig          *rest.Config
	KubeconfigPath      string
	EventuallyPoll      string
	EventuallyTimeout   string
	ConsistentlyPoll    string
	ConsistentlyTimeout string
}

// NewToolkitFromRest returns a toolkit using the given REST config. This is
// the preferred way to get a Toolkit instance, to avoid unset fields.
//
// The toolkit will use a new client built from the REST config and the global
// Scheme. The path to a kubeconfig can also be provided, which will be used
// for `.Kubectl` calls - if passed an empty string, a temporary kubeconfig
// will be created based on the REST config.
func NewToolkitFromRest(tkCfg *rest.Config, kubeconfigPath string) (Toolkit, error) {
	k8sClient, err := client.New(tkCfg, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		return Toolkit{}, err
	}

	// Create a temporary kubeconfig if one is not provided.
	if kubeconfigPath == "" {
		f, err := os.CreateTemp("", "toolkit-kubeconfig-*")
		if err != nil {
			return Toolkit{}, err
		}

		contents, err := createKubeconfigFile(tkCfg)
		if err != nil {
			return Toolkit{}, err
		}

		_, err = f.Write(contents)
		if err != nil {
			return Toolkit{}, err
		}

		kubeconfigPath = f.Name()
	}

	return Toolkit{
		Client:              k8sClient,
		Ctx:                 context.Background(),
		RestConfig:          tkCfg,
		KubeconfigPath:      kubeconfigPath,
		EventuallyPoll:      "100ms",
		EventuallyTimeout:   "1s",
		ConsistentlyPoll:    "100ms",
		ConsistentlyTimeout: "1s",
	}, nil
}

func (tk Toolkit) WithEPoll(eventuallyPoll string) Toolkit {
	tk.EventuallyPoll = eventuallyPoll

	return tk
}

func (tk Toolkit) WithETimeout(eventuallyTimeout string) Toolkit {
	tk.EventuallyTimeout = eventuallyTimeout

	return tk
}

func (tk Toolkit) WithCPoll(consistentlyPoll string) Toolkit {
	tk.ConsistentlyPoll = consistentlyPoll

	return tk
}

func (tk Toolkit) WithCTimeout(consistentlyTimeout string) Toolkit {
	tk.ConsistentlyTimeout = consistentlyTimeout

	return tk
}

func (tk Toolkit) WithCtx(ctx context.Context) Toolkit {
	tk.Ctx = ctx

	return tk
}

// CleanlyCreate creates the given object, and registers a callback to delete the object which
// Ginkgo will call at the appropriate time. The error from the `Create` call is returned (so it
// can be checked) and the `Delete` callback handles 'NotFound' errors as a success.
func (tk Toolkit) CleanlyCreate(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	createErr := tk.Create(ctx, obj, opts...)

	if createErr == nil {
		ginkgo.DeferCleanup(func() {
			ginkgo.GinkgoWriter.Printf("Deleting %v %v/%v\n",
				obj.GetObjectKind().GroupVersionKind().Kind, obj.GetNamespace(), obj.GetName())

			if err := tk.Delete(tk.Ctx, obj); err != nil {
				if !k8sErrors.IsNotFound(err) {
					// Use Fail in order to provide a custom message with useful information
					ginkgo.Fail(fmt.Sprintf("Expected success or 'NotFound' error, got %v", err), 1)
				}
			}
		})
	}

	return createErr
}

// Create uses the toolkit's client to save the object in the Kubernetes cluster.
// The only change in behavior is that it saves and restores the object's type
// information, which might otherwise be stripped during the API call.
func (tk Toolkit) Create(
	ctx context.Context, obj client.Object, opts ...client.CreateOption,
) error {
	savedGVK := obj.GetObjectKind().GroupVersionKind()
	err := tk.Client.Create(ctx, obj, opts...)
	obj.GetObjectKind().SetGroupVersionKind(savedGVK)

	return err
}

// Patch uses the toolkit's client to patch the object in the Kubernetes cluster.
// The only change in behavior is that it saves and restores the object's type
// information, which might otherwise be stripped during the API call.
func (tk Toolkit) Patch(
	ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption,
) error {
	savedGVK := obj.GetObjectKind().GroupVersionKind()
	err := tk.Client.Patch(ctx, obj, patch, opts...)
	obj.GetObjectKind().SetGroupVersionKind(savedGVK)

	return err
}

// Update uses the toolkit's client to update the object in the Kubernetes cluster.
// The only change in behavior is that it saves and restores the object's type
// information, which might otherwise be stripped during the API call.
func (tk Toolkit) Update(
	ctx context.Context, obj client.Object, opts ...client.UpdateOption,
) error {
	savedGVK := obj.GetObjectKind().GroupVersionKind()
	err := tk.Client.Update(ctx, obj, opts...)
	obj.GetObjectKind().SetGroupVersionKind(savedGVK)

	return err
}

// This regular expression is copied from
// https://github.com/open-cluster-management-io/governance-policy-framework-addon/blob/v0.13.0/controllers/statussync/policy_status_sync.go#L220
var compEventRegex = regexp.MustCompile(`(?i)^policy:\s*(?:([a-z0-9.-]+)\s*\/)?(.+)`) //nolint:gocritic // copy

// GetComplianceEvents queries the cluster and returns a sorted list of the Kubernetes
// compliance events for the given policy.
func (tk Toolkit) GetComplianceEvents(
	ctx context.Context, ns string, parentUID types.UID, templateName string,
) ([]corev1.Event, error) {
	list := &corev1.EventList{}

	err := tk.List(ctx, list, client.InNamespace(ns))
	if err != nil {
		return nil, err
	}

	events := make([]corev1.Event, 0)

	for i := range list.Items {
		event := list.Items[i]

		if event.InvolvedObject.UID != parentUID {
			continue
		}

		submatch := compEventRegex.FindStringSubmatch(event.Reason)
		if len(submatch) >= 3 && submatch[2] == templateName {
			events = append(events, event)
		}
	}

	sort.SliceStable(events, func(i, j int) bool {
		return events[i].Name < events[j].Name
	})

	return events, nil
}

// EC runs assertions on asynchronous behavior, both *E*ventually and *C*onsistently,
// using the polling and timeout settings of the toolkit. Its usage should feel familiar
// to gomega users, simply skip the `.Should(...)` call and put your matcher as the second
// parameter here.
func (tk Toolkit) EC(
	actualOrCtx interface{}, matcher gomegaTypes.GomegaMatcher, optionalDescription ...interface{},
) bool {
	ginkgo.GinkgoHelper()

	// Add where the failure occurred to the description
	eDesc := make([]interface{}, 1)
	cDesc := make([]interface{}, 1)

	//nolint:forcetypeassert // gomega makes the same unchecked assertions
	switch len(optionalDescription) {
	case 0:
		eDesc[0] = "Failed in Eventually"
		cDesc[0] = "Failed in Consistently"
	case 1:
		if origDescFunc, ok := optionalDescription[0].(func() string); ok {
			eDesc[0] = func() string {
				return "Failed in Eventually; " + origDescFunc()
			}
			cDesc[0] = func() string {
				return "Failed in Consistently; " + origDescFunc()
			}
		} else {
			eDesc[0] = "Failed in Eventually; " + optionalDescription[0].(string)
			cDesc[0] = "Failed in Consistently; " + optionalDescription[0].(string)
		}
	default:
		eDesc[0] = "Failed in Eventually; " + optionalDescription[0].(string)
		eDesc = append(eDesc, optionalDescription[1:]...) //nolint:makezero // appending is definitely correct

		cDesc[0] = "Failed in Consistently; " + optionalDescription[0].(string)
		cDesc = append(cDesc, optionalDescription[1:]...) //nolint:makezero // appending is definitely correct
	}

	gomega.Eventually(
		actualOrCtx, tk.EventuallyTimeout, tk.EventuallyPoll,
	).Should(matcher, eDesc...)

	return gomega.Consistently(
		actualOrCtx, tk.ConsistentlyTimeout, tk.ConsistentlyPoll,
	).Should(matcher, cDesc...)
}

func (tk *Toolkit) Kubectl(args ...string) (string, error) {
	addKubeconfig := true

	for _, arg := range args {
		if strings.HasPrefix(arg, "--kubeconfig") {
			addKubeconfig = false

			break
		}
	}

	if addKubeconfig {
		if tk.KubeconfigPath == "" {
			return "", ErrNoKubeconfg
		}

		args = append([]string{"--kubeconfig=" + tk.KubeconfigPath}, args...)
	}

	output, err := exec.Command("kubectl", args...).Output()

	var exitError *exec.ExitError

	if errors.As(err, &exitError) {
		if exitError.Stderr == nil {
			return string(output), err
		}

		return string(output), fmt.Errorf("%w: %s", ErrKubectl, exitError.Stderr)
	}

	return string(output), err
}

func createKubeconfigFile(cfg *rest.Config) ([]byte, error) {
	identifier := "toolkit"

	kubeconfig := api.NewConfig()

	cluster := api.NewCluster()
	cluster.Server = cfg.Host
	cluster.CertificateAuthorityData = cfg.CAData
	kubeconfig.Clusters[identifier] = cluster

	authInfo := api.NewAuthInfo()
	authInfo.ClientCertificateData = cfg.CertData
	authInfo.ClientKeyData = cfg.KeyData
	kubeconfig.AuthInfos[identifier] = authInfo

	apiContext := api.NewContext()
	apiContext.Cluster = identifier
	apiContext.AuthInfo = identifier
	kubeconfig.Contexts[identifier] = apiContext
	kubeconfig.CurrentContext = identifier

	return clientcmd.Write(*kubeconfig)
}
