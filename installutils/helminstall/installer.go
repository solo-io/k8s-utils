package helminstall

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/rotisserie/eris"
	"github.com/solo-io/k8s-utils/installutils/helminstall/types"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli/values"
	"helm.sh/helm/v3/pkg/getter"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"sigs.k8s.io/yaml"
)

//go:generate mockgen -destination mocks/mock_helm_installer.go -source ./installer.go

var (
	ReleaseAlreadyInstalledErr = func(name, namespace string) error {
		return eris.Errorf("The helm release you are trying to install (%s) appears"+
			" to already exist in %s", name, namespace)
	}
)

type installer struct {
	helmClient   types.HelmClient
	kubeNsClient NamespaceClient
	out          io.Writer
}

func NewInstaller(helmClient types.HelmClient, kubeNsClient NamespaceClient, outputWriter io.Writer) types.Installer {
	return &installer{
		helmClient:   helmClient,
		kubeNsClient: kubeNsClient,
		out:          outputWriter,
	}
}

func (i *installer) Install(ctx context.Context, installerConfig *types.InstallerConfig) error {
	namespace := installerConfig.InstallNamespace
	releaseName := installerConfig.ReleaseName
	if !installerConfig.DryRun {
		if releaseExists, err := i.helmClient.ReleaseExists(namespace, releaseName); err != nil {
			return err
		} else if releaseExists {
			return ReleaseAlreadyInstalledErr(releaseName, namespace)
		}
		if installerConfig.CreateNamespace {
			// Create the namespace if it doesn't exist. Helm3 no longer does this.
			i.createNamespace(ctx, namespace)
		}
	}

	if !installerConfig.DryRun && installerConfig.PreInstallMessage != "" {
		fmt.Fprint(i.out, installerConfig.PreInstallMessage)
	} else {
		i.defaultPreInstallMessage(installerConfig)
	}

	helmInstall, helmEnv, err := i.helmClient.NewInstall(namespace, releaseName, installerConfig.DryRun)
	if err != nil {
		return err
	}

	if installerConfig.Verbose {
		fmt.Printf("Looking for chart at %s\n", installerConfig.ReleaseUri)
	}

	chartObj, err := i.helmClient.DownloadChart(installerConfig.ReleaseUri)
	if err != nil {
		return err
	}

	// Merge values provided via the '--values' flag
	valueOpts := &values.Options{
		ValueFiles: installerConfig.ValuesFiles,
	}
	cliValues, err := valueOpts.MergeValues(getter.All(helmEnv))
	if err != nil {
		return err
	}

	// Merge the CLI flag values into the extra values, giving the latter higher precedence.
	// (The first argument to CoalesceTables has higher priority)
	completeValues := chartutil.CoalesceTables(installerConfig.ExtraValues, cliValues)
	if installerConfig.Verbose {
		b, err := json.Marshal(completeValues)
		if err != nil {
			fmt.Fprintf(i.out, "error: %v\n", err)
		}
		y, err := yaml.JSONToYAML(b)
		if err != nil {
			fmt.Fprintf(i.out, "error: %v\n", err)
		}
		fmt.Fprintf(i.out, "Installing the %s chart with the following value overrides:\n%s\n", chartObj.Metadata.Name, string(y))
	}

	rel, err := helmInstall.Run(chartObj, completeValues)
	if err != nil {
		return err
	}
	if !installerConfig.DryRun && installerConfig.PostInstallMessage != "" {
		fmt.Fprint(i.out, installerConfig.PostInstallMessage)
	} else {
		i.defaultPostInstallMessage(installerConfig)
	}

	if installerConfig.Verbose {
		fmt.Printf("Successfully ran helm install with release %s\n", releaseName)
	}

	if installerConfig.DryRun {
		fmt.Fprint(i.out, rel.Manifest)
	}

	return nil
}

func (i *installer) createNamespace(ctx context.Context, namespace string) {
	_, err := i.kubeNsClient.Get(ctx, namespace, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		fmt.Fprintf(i.out, "Creating namespace %s... ", namespace)
		if _, err := i.kubeNsClient.Create(ctx, &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		}); err != nil {
			fmt.Fprintf(i.out, "\nUnable to create namespace %s (%s). Continuing...\n", namespace, err.Error())
		} else {
			fmt.Fprintf(i.out, "Done.\n")
		}
	} else if err != nil {
		fmt.Fprintf(i.out, "\nUnable to check if namespace %s exists (%s). Continuing...\n", namespace, err.Error())
	}

}

func (i *installer) defaultPreInstallMessage(config *types.InstallerConfig) {
	if config.DryRun {
		return
	}
	fmt.Fprintf(i.out, "Starting helm installation\n")
}

func (i *installer) defaultPostInstallMessage(config *types.InstallerConfig) {
	if config.DryRun {
		return
	}
	fmt.Fprintf(i.out, "Successful installation!\n")
}

type NamespaceClient interface {
	Create(ctx context.Context, ns *corev1.Namespace) (*corev1.Namespace, error)
	Delete(ctx context.Context, name string, options *metav1.DeleteOptions) error
	Get(ctx context.Context, name string, options metav1.GetOptions) (*corev1.Namespace, error)
	List(ctx context.Context, opts metav1.ListOptions) (*corev1.NamespaceList, error)
}

type namespaceClient struct {
	client v1.NamespaceInterface
}

func (n *namespaceClient) Create(ctx context.Context, ns *corev1.Namespace) (*corev1.Namespace, error) {
	return n.client.Create(ctx, ns, metav1.CreateOptions{})
}

func (n *namespaceClient) Delete(ctx context.Context, name string, options *metav1.DeleteOptions) error {
	opts := metav1.DeleteOptions{}
	if options != nil {
		opts = *options
	}
	return n.client.Delete(ctx, name, opts)
}

func (n *namespaceClient) Get(ctx context.Context, name string, options metav1.GetOptions) (*corev1.Namespace, error) {
	return n.client.Get(ctx, name, options)
}

func (n *namespaceClient) List(ctx context.Context, opts metav1.ListOptions) (*corev1.NamespaceList, error) {
	return n.List(ctx, opts)
}

func NewNamespaceClient(client v1.NamespaceInterface) *namespaceClient {
	return &namespaceClient{client: client}
}
