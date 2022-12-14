package kubeutils

// SoloClusterAnnotation was originally implemented in solo-kitv2's ezkube package
// It allows for easy mimicking of the old clustername field present in
// k8s prior to 1.24. It is stored here to make future changes easier and not
// require multi package switches as if any of the dependant packages are out of
// sync on this field it can cause hard to diagnose issues.
const SoloClusterAnnotation = "cluster.solo.io/cluster"

type annotationStore interface {
	SetAnnotations(map[string]string)
	GetAnnotations() map[string]string
}

// GetClusterName from within the annotations
func GetClusterName(as annotationStore) string {
	return as.GetAnnotations()[SoloClusterAnnotation]
}

// SetClusterName on the retrieved annotations
// Set annotations which while slow is correct.
func SetClusterName(as annotationStore, clusterName string) {
	anno := as.GetAnnotations()
	if anno == nil {
		anno = map[string]string{}
	}
	anno[SoloClusterAnnotation] = clusterName
	as.SetAnnotations(anno)
}
