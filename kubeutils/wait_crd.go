package kubeutils

import (
	"context"
	"time"

	apiv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	"github.com/avast/retry-go"
	"github.com/pkg/errors"
	"github.com/rotisserie/eris"
	apiexts "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Waits for a CRD to be "established" in kubernetes, which means it's active an can be
// CRUD'ed by clients
func WaitForCrdActive(ctx context.Context, apiexts apiexts.Interface, crdName string) error {
	return retry.Do(func() error {
		crd, err := apiexts.ApiextensionsV1().CustomResourceDefinitions().Get(ctx, crdName, metav1.GetOptions{})
		if err != nil {
			return errors.Wrapf(err, "lookup crd %v", crdName)
		}

		var established bool
		for _, status := range crd.Status.Conditions {
			if status.Type == apiv1.Established {
				established = true
				break
			}
		}

		if !established {
			return eris.Errorf("crd %v exists but not yet established by kube", crdName)
		}

		return nil
	},
		retry.Delay(time.Millisecond*500),
		retry.DelayType(retry.FixedDelay),
	)
}
