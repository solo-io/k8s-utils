package kubeinstallutils

import (
	"context"

	apiv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	"github.com/rotisserie/eris"
	apiexts "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CrdsFromManifest(crdManifestYaml string) ([]*apiv1.CustomResourceDefinition, error) {
	var crds []*apiv1.CustomResourceDefinition
	crdRuntimeObjects, err := ParseKubeManifest(crdManifestYaml)
	if err != nil {
		return nil, err
	}
	for _, obj := range crdRuntimeObjects {
		apiExtCrd, ok := obj.(*apiv1.CustomResourceDefinition)
		if !ok {
			return nil, eris.Wrap(err, "internal error: crd manifest must only contain CustomResourceDefinitions")
		}
		crds = append(crds, apiExtCrd)
	}
	return crds, nil
}

func CreateCrds(ctx context.Context, apiExts apiexts.Interface, crds ...*apiv1.CustomResourceDefinition) error {
	for _, crd := range crds {
		if _, err := apiExts.ApiextensionsV1().CustomResourceDefinitions().Create(ctx, crd, v1.CreateOptions{}); err != nil && !apierrors.IsAlreadyExists(err) {
			return eris.Wrapf(err, "failed to create crd: %v", crd)
		}
	}
	return nil
}

func DeleteCrds(ctx context.Context, apiExts apiexts.Interface, crdNames ...string) error {
	for _, name := range crdNames {
		err := apiExts.ApiextensionsV1().CustomResourceDefinitions().Delete(ctx, name, v1.DeleteOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			return eris.Wrapf(err, "failed to delete crd: %v", name)
		}
	}
	return nil
}
