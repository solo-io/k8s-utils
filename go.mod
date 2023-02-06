module github.com/solo-io/k8s-utils

go 1.19

require (
	github.com/avast/retry-go v2.2.0+incompatible
	github.com/evanphx/json-patch v4.12.0+incompatible
	github.com/ghodss/yaml v1.0.0
	github.com/gogo/protobuf v1.3.2
	github.com/golang/mock v1.6.0
	github.com/golang/protobuf v1.5.2
	github.com/google/go-github/v32 v32.0.0
	github.com/google/uuid v1.2.0
	github.com/goph/emperror v0.17.1
	github.com/hashicorp/consul/api v1.1.0
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.24.0
	github.com/pkg/errors v0.9.1
	github.com/rotisserie/eris v0.1.1
	github.com/solo-io/go-utils v0.22.4
	github.com/spf13/afero v1.6.0
	go.uber.org/zap v1.19.0
	golang.org/x/sync v0.0.0-20220722155255-886fb9371eb4
	google.golang.org/protobuf v1.28.0
	helm.sh/helm/v3 v3.9.0
	k8s.io/api v0.25.4
	k8s.io/apiextensions-apiserver v0.25.4
	k8s.io/apimachinery v0.25.4
	k8s.io/cli-runtime v0.24.0
	k8s.io/client-go v0.25.4
	sigs.k8s.io/controller-runtime v0.6.0
	sigs.k8s.io/yaml v1.3.0
)

require (
	github.com/MakeNowJust/heredoc v0.0.0-20171113091838-e9091a26100e // indirect
	github.com/bugsnag/bugsnag-go v1.5.0 // indirect
	github.com/docker/libtrust v0.0.0-20160708172513-aabc10ec26b7 // indirect
	github.com/onsi/ginkgo/v2 v2.5.0
	github.com/yvasiyarov/go-metrics v0.0.0-20150112132944-c25f46c4b940 // indirect
	github.com/yvasiyarov/gorelic v0.0.6 // indirect
)

replace (
	// logrus did the rename of their repo which is why we have this
	github.com/Sirupsen/logrus => github.com/sirupsen/logrus v1.0.5

	// pin to the jwt-go fork to fix CVE.
	// using the pseudo version of github.com/form3tech-oss/jwt-go@v3.2.3 instead of the version directly,
	// to avoid error about it being used for two different module paths
	github.com/dgrijalva/jwt-go => github.com/form3tech-oss/jwt-go v0.0.0-20210511163231-5b2d2b5f6c34
	github.com/docker/docker => github.com/moby/moby v0.7.3-0.20190826074503-38ab9da00309
)
