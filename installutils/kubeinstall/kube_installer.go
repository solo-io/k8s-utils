package kubeinstall

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	apiv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	"github.com/pkg/errors"
	"github.com/solo-io/k8s-utils/kubeerrutils"

	"go.uber.org/zap"
	batchv1 "k8s.io/api/batch/v1"

	"github.com/solo-io/go-utils/stringutils"

	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	"github.com/avast/retry-go"
	"github.com/rotisserie/eris"
	"github.com/solo-io/go-utils/contextutils"
	"github.com/solo-io/k8s-utils/installutils/kuberesource"
	"golang.org/x/sync/errgroup"
	appsv1 "k8s.io/api/apps/v1"
	appsv1beta2 "k8s.io/api/apps/v1beta2"
	kubev1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiexts "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	kubeerrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// an interface allowing these methods to be mocked
type Installer interface {
	ReconcileResources(ctx context.Context, params ReconcileParams) error
	PurgeResources(ctx context.Context, withLabels map[string]string) error
	ListAllResources(ctx context.Context) kuberesource.UnstructuredResources
}

type KubeInstaller struct {
	cache          *Cache
	cfg            *rest.Config
	dynamic        dynamic.Interface
	client         client.Client
	core           kubernetes.Interface
	apiExtensions  apiexts.Interface
	callbacks      []CallbackOptions
	retryOptions   []retry.Option
	creationPolicy CreationPolicy
}

var _ Installer = &KubeInstaller{}

// policies for how to handle resource creation
// specifies behavior when resource creation fails
type CreationPolicy int

const (
	// Attempt to create, return any error
	CreationPolicy_ReturnErrors = iota
	// Attempt to create, ignore  AlreadyExists error
	CreationPolicy_IgnoreOnExists
	// Attempt to create, fall back to Update on AlreadyExists error
	CreationPolicy_UpdateOnExists
	// Attempt to create, fall back to Update on AlreadyExists error, fall back to Destroy&Recreate on ImmutableField error
	CreationPolicy_ForceUpdateOnExists
)

type KubeInstallerOptions struct {
	Callbacks    []CallbackOptions
	RetryOptions []retry.Option
	// define how to handle AlreadyExist errors on resource creation
	CreationPolicy CreationPolicy
}

var defaultRetryOptions = []retry.Option{
	retry.Delay(time.Millisecond * 250),
	retry.DelayType(retry.FixedDelay),
	retry.Attempts(500), // give a considerable amount of time for pulling images
}

/*
NewKubeInstaller does not initialize the cache.
Should be one once globally
*/
func NewKubeInstaller(cfg *rest.Config, cache *Cache, opts *KubeInstallerOptions) (*KubeInstaller, error) {

	apiExts, err := apiexts.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	dynamicClient, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	core, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	client, err := client.New(cfg, client.Options{})
	if err != nil {
		return nil, err
	}

	callbacks := initCallbacks()
	retryOpts := defaultRetryOptions

	var (
		creationPolicy CreationPolicy
	)
	if opts != nil {
		for _, cb := range opts.Callbacks {
			callbacks = append(callbacks, cb)
		}
		if len(opts.RetryOptions) > 0 {
			retryOpts = opts.RetryOptions
		}
		creationPolicy = opts.CreationPolicy
	}

	return &KubeInstaller{
		cache:          cache,
		cfg:            cfg,
		apiExtensions:  apiExts,
		client:         client,
		dynamic:        dynamicClient,
		core:           core,
		callbacks:      callbacks,
		retryOptions:   retryOpts,
		creationPolicy: creationPolicy,
	}, nil
}

func (r *KubeInstaller) preInstall() error {
	for _, cb := range r.callbacks {
		if err := cb.PreInstall(); err != nil {
			return errors.Wrapf(err, "error in pre-install hook")
		}
	}
	return nil
}

func (r *KubeInstaller) postInstall() error {
	for _, cb := range r.callbacks {
		if err := cb.PostInstall(); err != nil {
			return errors.Wrapf(err, "error in post-install hook")
		}
	}
	return nil
}

func (r *KubeInstaller) preCreate(res *unstructured.Unstructured) error {
	for _, cb := range r.callbacks {
		if err := cb.PreCreate(res); err != nil {
			return errors.Wrapf(err, "error in pre-create hook")
		}
	}
	return nil
}

func (r *KubeInstaller) postCreate(res *unstructured.Unstructured) error {
	for _, cb := range r.callbacks {
		if err := cb.PostCreate(res); err != nil {
			return errors.Wrapf(err, "error in post-create hook")
		}
	}
	return nil
}

func (r *KubeInstaller) preUpdate(res *unstructured.Unstructured) error {
	if err := setInstallationAnnotation(res); err != nil {
		return err
	}
	for _, cb := range r.callbacks {
		if err := cb.PreUpdate(res); err != nil {
			return errors.Wrapf(err, "error in pre-update hook")
		}
	}
	return nil
}

func (r *KubeInstaller) postUpdate(res *unstructured.Unstructured) error {
	for _, cb := range r.callbacks {
		if err := cb.PostUpdate(res); err != nil {
			return errors.Wrapf(err, "error in post-update hook")
		}
	}
	return nil
}

func (r *KubeInstaller) preDelete(res *unstructured.Unstructured) error {
	for _, cb := range r.callbacks {
		if err := cb.PreDelete(res); err != nil {
			return errors.Wrapf(err, "error in pre-delete hook")
		}
	}
	return nil
}

func (r *KubeInstaller) postDelete(res *unstructured.Unstructured) error {
	for _, cb := range r.callbacks {
		if err := cb.PostDelete(res); err != nil {
			return errors.Wrapf(err, "error in post-delete hook")
		}
	}
	return nil
}

type ReconcileParams struct {
	InstallNamespace string
	Resources        kuberesource.UnstructuredResources
	OwnerLabels      map[string]string
	// respect hard-coded namespaces inside of manifests
	RespectManifestNamespaces bool
}

func NewReconcileParams(installNamespace string, resources kuberesource.UnstructuredResources, ownerLabels map[string]string, respectManifestNamespaces bool) ReconcileParams {
	return ReconcileParams{InstallNamespace: installNamespace, Resources: resources, OwnerLabels: ownerLabels, RespectManifestNamespaces: respectManifestNamespaces}
}

func (r *KubeInstaller) ReconcileResources(ctx context.Context, params ReconcileParams) error {
	if err := r.preInstall(); err != nil {
		return errors.Wrapf(err, "error in pre-install hook")
	}

	if err := r.reconcileResources(ctx, params.InstallNamespace, params.Resources, params.OwnerLabels, params.RespectManifestNamespaces); err != nil {
		return err
	}

	if err := r.postInstall(); err != nil {
		return errors.Wrapf(err, "error in pre-install hook")
	}

	return nil
}

const installerAnnotationKey = "installer.solo.io/last-applied-configuration"

// sets the installation annotation so we can do proper comparison on our objects
func setInstallationAnnotation(res *unstructured.Unstructured) error {
	jsn, err := json.Marshal(res)
	if err != nil {
		return err
	}

	annotations := res.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	annotations[installerAnnotationKey] = string(jsn)
	res.SetAnnotations(annotations)
	return nil
}

// attempts to get the installed version of the resource from the cache annotation key
// if it's not present, return the original object
func getInstalledResources(resources kuberesource.UnstructuredResources) (kuberesource.UnstructuredResources, error) {
	var installed kuberesource.UnstructuredResources
	for _, res := range resources {
		res, err := getInstalledResource(res)
		if err != nil {
			return nil, err
		}
		installed = append(installed, res)
	}
	return installed, nil
}

func getInstalledResource(res *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	installedConfiguration, ok := res.GetAnnotations()[installerAnnotationKey]
	if !ok {
		return nil, eris.Errorf("resource %v missing installer annotation %v", kuberesource.Key(res), installerAnnotationKey)
	}
	var installedObject map[string]interface{}
	if err := json.Unmarshal([]byte(installedConfiguration), &installedObject); err != nil {
		return nil, err
	}
	res.Object = installedObject
	annotations := res.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	annotations[installerAnnotationKey] = installedConfiguration
	res.SetAnnotations(annotations)
	return res, nil
}

func (r *KubeInstaller) reconcileResources(ctx context.Context, installNamespace string, desiredResources kuberesource.UnstructuredResources, ownerLabels map[string]string, respectManifestNamespaces bool) error {
	cachedResourceList, err := getInstalledResources(r.cache.List().WithLabels(ownerLabels))
	if err != nil {
		return err
	}
	cachedResources := cachedResourceList.ByKey()

	logger := contextutils.LoggerFrom(ctx)

	logger.Infow("reconciling desired resources against cached resources",
		"desired", len(desiredResources),
		"cached_with_label", len(cachedResources),
		"labels", ownerLabels,
		"cache_total", len(r.cache.resources),
	)

	httpClient := http.Client{}
	restMapper, err := apiutil.NewDynamicRESTMapper(r.cfg, &httpClient)
	if err != nil {
		return errors.Wrapf(err, "creating discovery rest mapper")
	}

	// refresh the client to get the new rest mappings for any crds created in the background (i.e. by a Job) since the client was last refreshed
	r.client, err = client.New(r.cfg, client.Options{})
	if err != nil {
		return err
	}

	// set labels for writing
	for _, res := range desiredResources {
		labels := res.GetLabels()
		if labels == nil {
			labels = make(map[string]string)
		}
		for k, v := range ownerLabels {
			labels[k] = v
		}
		res.SetLabels(labels)

		isNamespaced, err := r.isNamespaced(restMapper, desiredResources, kuberesource.Key(res))
		if err != nil {
			return err
		}
		if !respectManifestNamespaces {
			if isNamespaced {
				res.SetNamespace(installNamespace)
			} else {
				res.SetNamespace("")
			}
		}
	}

	desiredResourcesByKey := desiredResources.ByKey()

	// determine what must be created, deleted, updated
	var resourcesToDelete, resourcesToCreate, resourcesToUpdate kuberesource.UnstructuredResources
	for key, res := range desiredResourcesByKey {
		if _, exists := cachedResources[key]; exists {
			resourcesToUpdate = append(resourcesToUpdate, res)
		} else {
			resourcesToCreate = append(resourcesToCreate, res)
		}
	}
	for key, res := range cachedResources {
		if _, desired := desiredResourcesByKey[key]; !desired {
			resourcesToDelete = append(resourcesToDelete, res)
		}
	}

	logger.Infof("preparing to create %v, update %v, and delete %v resources", len(resourcesToCreate), len(resourcesToUpdate), len(resourcesToDelete))

	// delete in reverse order of install
	groupedResourcesToDelete := resourcesToDelete.GroupedByGVK()
	for i := len(groupedResourcesToDelete); i > 0; i-- {
		group := groupedResourcesToDelete[i-1]
		g := errgroup.Group{}
		for _, res := range group.Resources {
			res := res
			g.Go(func() error {
				if err := r.preDelete(res); err != nil {
					return err
				}
				resKey := fmt.Sprintf("%v %v.%v", res.GroupVersionKind().Kind, res.GetNamespace(), res.GetName())
				logger.Infof("deleting resource %v", resKey)

				if err := retry.Do(func() error {
					return r.client.Delete(ctx, res.DeepCopy(), &deleteOptionsApplier{})
				}); err != nil && !kubeerrs.IsNotFound(err) {
					return errors.Wrapf(err, "deleting  %v", resKey)
				}
				r.cache.Delete(res)
				if err := r.postDelete(res); err != nil {
					return err
				}
				return nil
			})
		}
		if err := g.Wait(); err != nil {
			return err
		}
	}

	// create
	// ensure ns exists before performing a create
	if len(resourcesToCreate) > 0 {
		if _, err := r.core.CoreV1().Namespaces().Create(ctx, &kubev1.Namespace{
			ObjectMeta: v1.ObjectMeta{Name: installNamespace},
		}, v1.CreateOptions{}); err != nil && !kubeerrutils.IsAlreadyExists(err) {
			return errors.Wrapf(err, "creating installation namespace")
		}
	}
	for _, group := range resourcesToCreate.GroupedByGVK() {
		// batch create for each resource group
		g := errgroup.Group{}
		for _, res := range group.Resources {
			res := res
			g.Go(func() error {
				if err := r.preCreate(res); err != nil {
					return err
				}
				resKey := fmt.Sprintf("%v %v.%v", res.GroupVersionKind().Kind, res.GetNamespace(), res.GetName())
				logger.Infof("creating resource %v", resKey)

				if err := retry.Do(r.getCreationFunction(ctx, res)); err != nil {
					return errors.Wrapf(err, "creating %v", resKey)
				}
				r.cache.Set(res)
				if err := r.postCreate(res); err != nil {
					return err
				}
				if err := r.waitForResourceReady(ctx, res); err != nil {
					return errors.Wrapf(err, "waiting for resource to become ready %v", resKey)
				}
				return nil
			})
		}
		if err := g.Wait(); err != nil {
			return err
		}
	}

	// update
	for _, group := range resourcesToUpdate.GroupedByGVK() {
		g := errgroup.Group{}
		for _, res := range group.Resources {
			desired := res
			g.Go(func() error {
				if err := r.preUpdate(desired); err != nil {
					return err
				}
				key := kuberesource.Key(desired)
				original, ok := cachedResources[key]
				if !ok {
					return eris.Errorf("internal error: could not find original resource for desired key %v", key)
				}
				// don't update the object if there is a match
				if kuberesource.Match(ctx, original, desired) {
					return nil
				}
				patchedServerResource, err := r.patchServerResource(ctx, original, desired)
				if err != nil {
					return err
				}
				resKey := fmt.Sprintf("%v %v.%v", desired.GroupVersionKind().Kind, desired.GetNamespace(), desired.GetName())
				logger.Infof("updating resource %v", resKey)

				if err := retry.Do(func() error { return r.client.Update(ctx, patchedServerResource) }); err != nil {
					return errors.Wrapf(err, "updating %v", resKey)
				}
				r.cache.Set(desired)
				if err := r.waitForResourceReady(ctx, desired); err != nil {
					return errors.Wrapf(err, "waiting for resource to become ready %v", resKey)
				}
				return nil
			})
		}
		if err := g.Wait(); err != nil {
			return err
		}
	}

	logger.Infof("created %v, updated %v, and deleted %v resources", len(resourcesToCreate), len(resourcesToUpdate), len(resourcesToDelete))

	return nil
}

func (r *KubeInstaller) isNamespaced(restMapper meta.RESTMapper, desiredResources kuberesource.UnstructuredResources, key kuberesource.ResourceKey) (bool, error) {
	mapping, err := restMapper.RESTMapping(key.Gvk.GroupKind(), key.Gvk.Version)
	if err != nil {
		if !meta.IsNoMatchError(err) {
			return false, err
		}

		// resource might be an unregistered Custom Resource
		// try to determine whether the desired object should be namespaced based on the CRD spec with a lookup
		var isNamespaced bool
		crdResource := desiredResources.Filter(func(resource *unstructured.Unstructured) bool {
			runtimeObj, err := kuberesource.ConvertUnstructured(resource)
			if err != nil {
				return true
			}
			crd, ok := runtimeObj.(*apiextensions.CustomResourceDefinition)
			if !ok {
				return true
			}
			if crd.Spec.Group == key.Gvk.Group && key.Gvk.Version == crd.Spec.Version && key.Gvk.Kind == crd.Spec.Names.Kind {
				isNamespaced = crd.Spec.Scope == apiextensions.NamespaceScoped
				return false // filter all except this crd
			}
			return true
		})

		if len(crdResource) != 1 {
			return false, errors.Wrapf(err, "could not get rest mapping and could not find crd for %v", key)
		}

		return isNamespaced, nil

	}
	return mapping.Scope.Name() != meta.RESTScopeNameRoot, nil
}

func (r *KubeInstaller) getCreationFunction(ctx context.Context, res *unstructured.Unstructured) func() error {

	resCopy := res.DeepCopy()

	switch r.creationPolicy {
	default:
		return func() error {
			return r.client.Create(ctx, res.DeepCopy())
		}
	case CreationPolicy_IgnoreOnExists:
		return func() error {
			// create, only return err if !AlreadyExists
			if err := r.client.Create(ctx, resCopy); err != nil && !kubeerrutils.IsAlreadyExists(err) {
				return err
			}
			return nil
		}
	case CreationPolicy_UpdateOnExists:
		return func() error {
			// create, return if success or non AlreadyExists err occurred
			if err := r.client.Create(ctx, resCopy); err == nil || !kubeerrutils.IsAlreadyExists(err) {
				return err
			}
			if err := r.updateResourceVersion(ctx, resCopy); err != nil {
				return err
			}
			// attempt update
			return r.client.Update(ctx, resCopy)
		}
	case CreationPolicy_ForceUpdateOnExists:
		return func() error {
			// create, return if success or non AlreadyExists err occurred
			if err := r.client.Create(ctx, resCopy); err == nil || !kubeerrutils.IsAlreadyExists(err) {
				return err
			}
			if err := r.updateResourceVersion(ctx, resCopy); err != nil {
				return err
			}
			// attempt update, return if success or non Immutability error occurred
			if err := r.client.Update(ctx, resCopy); err == nil || !kubeerrutils.IsImmutableErr(err) {
				return err
			}
			if err := r.client.Delete(ctx, resCopy); err != nil {
				return err
			}
			if err := r.waitForNotExist(ctx, resCopy); err != nil {
				return err
			}
			resCopy.SetResourceVersion("")

			return r.client.Create(ctx, resCopy)
		}
	}
}

// create a patch from the diff between our cached object and the desired resource
// then apply that patch to the server's current version fo the resource
func (r *KubeInstaller) patchServerResource(ctx context.Context, original, desired *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	currentFromServer := original.DeepCopyObject().(*unstructured.Unstructured)
	objectKey := client.ObjectKey{Namespace: original.GetNamespace(), Name: original.GetName()}
	if err := r.client.Get(ctx, objectKey, currentFromServer); err != nil {
		return nil, err
	}

	patch, err := kuberesource.GetPatch(original, desired)
	if err != nil {
		return nil, err
	}

	if err := kuberesource.Patch(currentFromServer, patch); err != nil {
		return nil, err
	}
	return currentFromServer, nil
}

// do an HTTP GET to update the resource version of the desired object
func (r *KubeInstaller) updateResourceVersion(ctx context.Context, res *unstructured.Unstructured) error {
	currentFromServer := res.DeepCopyObject().(*unstructured.Unstructured)
	objectKey := client.ObjectKey{Namespace: res.GetNamespace(), Name: res.GetName()}
	if err := r.client.Get(ctx, objectKey, currentFromServer); err != nil {
		return err
	}
	res.SetResourceVersion(currentFromServer.GetResourceVersion())
	return nil
}

func (r *KubeInstaller) PurgeResources(ctx context.Context, withLabels map[string]string) error {
	return r.reconcileResources(ctx, "", nil, withLabels, false)
}

func (r *KubeInstaller) ListAllResources(ctx context.Context) kuberesource.UnstructuredResources {
	return r.cache.List()
}

func ListAllCachedValues(ctx context.Context, labelKey string, installer Installer) []string {
	var values []string
	for _, res := range installer.ListAllResources(ctx) {
		value := res.GetLabels()[labelKey]
		if value != "" && !stringutils.ContainsString(value, values) {
			values = append(values, value)
		}
	}
	return values
}

func (r *KubeInstaller) waitForResourceReady(ctx context.Context, res *unstructured.Unstructured) error {
	runtimeObject, err := kuberesource.ConvertUnstructured(res)
	if err != nil {
		return nil // not a handled type, possibly a crd
	}
	switch obj := runtimeObject.(type) {
	case *v1beta1.CustomResourceDefinition:
		if err := r.waitForCrd(ctx, obj.Name); err != nil {
			return err
		}
		// refresh the client to get the new rest mappings for the crd
		r.client, err = client.New(r.cfg, client.Options{})
		if err != nil {
			return err
		}
	case *extensionsv1beta1.Deployment:
		return r.waitForDeploymentReplica(ctx, obj.Name, obj.Namespace)
	case *appsv1.Deployment:
		return r.waitForDeploymentReplica(ctx, obj.Name, obj.Namespace)
	case *appsv1beta2.Deployment:
		return r.waitForDeploymentReplica(ctx, obj.Name, obj.Namespace)
	case *batchv1.Job:
		return r.waitForJobComplete(ctx, obj.Name, obj.Namespace)
	}
	return nil
}

func (r *KubeInstaller) waitForCrd(ctx context.Context, crdName string) error {
	return retry.Do(func() error {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		crd, err := r.apiExtensions.ApiextensionsV1().CustomResourceDefinitions().Get(ctx, crdName, v1.GetOptions{})
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

		// attempt to do a list on the crd's resources. the above can still give false positives
		_, err = r.dynamic.Resource(schema.GroupVersionResource{
			Group:    crd.Spec.Group,
			Version:  crd.Spec.Versions[0].Name,
			Resource: crd.Spec.Names.Plural,
		}).List(ctx, v1.ListOptions{})
		if err != nil {
			return err
		}

		contextutils.LoggerFrom(ctx).Infow("registered crd", zap.Any("name", crd.ObjectMeta.Name))

		return nil
	},
		r.retryOptions...,
	)
}

func (r *KubeInstaller) waitForDeploymentReplica(ctx context.Context, name, namespace string) error {
	return retry.Do(func() error {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		deployment, err := r.core.AppsV1().Deployments(namespace).Get(ctx, name, v1.GetOptions{})
		if err != nil {
			return errors.Wrapf(err, "lookup deployment %v.%v", name, namespace)
		}

		// no replicas to wait for
		if deployment.Spec.Replicas != nil && *deployment.Spec.Replicas == 0 {
			return nil
		}

		// wait for at least one replica to become ready
		if deployment.Status.ReadyReplicas < 1 {
			var condition appsv1.DeploymentCondition
			if len(deployment.Status.Conditions) > 0 {
				condition = deployment.Status.Conditions[0]
			}
			return eris.Errorf("no ready replicas for deployment %v.%v with condition %#v", namespace, name,
				condition)
		}

		contextutils.LoggerFrom(ctx).Infof("deployment %v.%v ready", namespace, name)
		return nil
	},
		r.retryOptions...,
	)
}

func (r *KubeInstaller) waitForJobComplete(ctx context.Context, name, namespace string) error {
	return retry.Do(func() error {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		job, err := r.core.BatchV1().Jobs(namespace).Get(ctx, name, v1.GetOptions{})
		if err != nil {
			return errors.Wrapf(err, "lookup job %v.%v", name, namespace)
		}

		// Wait for completion time to be set and a condition of type "Complete"
		// per completeness definition in https://github.com/kubernetes/kubernetes/issues/68712#issuecomment-514008330
		if job.Status.CompletionTime != nil {
			for _, condition := range job.Status.Conditions {
				if condition.Type == batchv1.JobComplete {
					contextutils.LoggerFrom(ctx).Infof("job %v.%v complete", namespace, name)
					return nil
				}
			}
		}

		// wait for at least one complete run
		var condition batchv1.JobCondition
		if len(job.Status.Conditions) > 0 {
			condition = job.Status.Conditions[0]
		}
		return eris.Errorf("no successful runs of job %v.%v with condition %#v", namespace, name, condition)
	},
		r.retryOptions...,
	)
}

func (r *KubeInstaller) waitForNotExist(ctx context.Context, res *unstructured.Unstructured) error {
	return retry.Do(func() error {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		objectKey := client.ObjectKey{Namespace: res.GetNamespace(), Name: res.GetName()}
		if err := r.client.Get(ctx, objectKey, res); err == nil {
			return eris.Errorf("resource %v still exists", res.GetName())
		} else if !kubeerrs.IsNotFound(err) {
			return err
		}
		return nil
	},
		r.retryOptions...,
	)
}

type deleteOptionsApplier struct {
}

func (d *deleteOptionsApplier) ApplyToDelete(options *client.DeleteOptions) {
	p := v1.DeletePropagationForeground
	options.PropagationPolicy = &p
}
