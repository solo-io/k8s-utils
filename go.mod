module github.com/solo-io/k8s-utils

go 1.14

require (
	github.com/MakeNowJust/heredoc v0.0.0-20171113091838-e9091a26100e // indirect
	github.com/avast/retry-go v2.2.0+incompatible
	github.com/bugsnag/bugsnag-go v1.5.0 // indirect
	github.com/docker/go-metrics v0.0.0-20181218153428-b84716841b82 // indirect
	github.com/docker/libtrust v0.0.0-20160708172513-aabc10ec26b7 // indirect
	github.com/docker/spdystream v0.0.0-20181023171402-6480d4af844c // indirect
	github.com/emicklei/go-restful/v3 v3.8.0 // indirect
	github.com/evanphx/json-patch v4.5.0+incompatible
	github.com/garyburd/redigo v1.6.0 // indirect
	github.com/ghodss/yaml v1.0.0
	github.com/go-logr/zapr v0.1.1 // indirect
	github.com/go-openapi/spec v0.19.4 // indirect
	github.com/gogo/protobuf v1.3.1
	github.com/golang/mock v1.4.4
	github.com/golang/protobuf v1.4.2
	github.com/google/go-github/v32 v32.0.0
	github.com/google/uuid v1.1.1
	github.com/goph/emperror v0.17.1
	github.com/gorilla/handlers v1.4.0 // indirect
	github.com/hashicorp/consul/api v1.1.0
	github.com/onsi/ginkgo v1.12.1
	github.com/onsi/gomega v1.10.1
	github.com/pkg/errors v0.9.1
	github.com/rotisserie/eris v0.1.1
	github.com/solo-io/go-utils v0.21.6
	github.com/spf13/afero v1.2.2
	github.com/xenolf/lego v0.3.2-0.20160613233155-a9d8cec0e656 // indirect
	github.com/yvasiyarov/go-metrics v0.0.0-20150112132944-c25f46c4b940 // indirect
	github.com/yvasiyarov/gorelic v0.0.6 // indirect
	go.uber.org/zap v1.10.0
	golang.org/x/sync v0.0.0-20200625203802-6e8e738ad208
	google.golang.org/protobuf v1.25.0
	gopkg.in/square/go-jose.v1 v1.1.2 // indirect
	helm.sh/helm/v3 v3.2.4
	k8s.io/api v0.18.6
	k8s.io/apiextensions-apiserver v0.18.6
	k8s.io/apimachinery v0.18.6
	k8s.io/cli-runtime v0.18.0
	k8s.io/client-go v0.18.6
	rsc.io/letsencrypt v0.0.1 // indirect
	sigs.k8s.io/controller-runtime v0.6.2
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.0.0+incompatible
	github.com/Sirupsen/logrus => github.com/sirupsen/logrus v1.0.5

	// pin to the jwt-go fork to fix CVE.
	// using the pseudo version of github.com/form3tech-oss/jwt-go@v3.2.3 instead of the version directly,
	// to avoid error about it being used for two different module paths
	github.com/dgrijalva/jwt-go => github.com/form3tech-oss/jwt-go v0.0.0-20210511163231-5b2d2b5f6c34
	github.com/docker/docker => github.com/moby/moby v0.7.3-0.20190826074503-38ab9da00309
)
