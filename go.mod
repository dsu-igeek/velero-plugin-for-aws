module github.com/vmware-tanzu/velero-plugin-for-aws

go 1.13

require (
	github.com/aws/aws-sdk-go v1.36.3
	github.com/hashicorp/go-hclog v0.9.2 // indirect
	github.com/hashicorp/go-plugin v1.0.1-0.20190610192547-a1bc61569a26 // indirect
	github.com/hashicorp/yamux v0.0.0-20190923154419-df201c70410d // indirect
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.7.0
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.6.1
	github.com/vmware-tanzu/astrolabe v0.0.0-00010101000000-000000000000
	github.com/vmware-tanzu/velero v1.4.0
	golang.org/x/tools v0.0.0-20201224043029-2b0845dc783e
	k8s.io/api v0.19.7
	k8s.io/apimachinery v0.19.7
)

replace github.com/vmware-tanzu/astrolabe => ../astrolabe

replace github.com/vmware-tanzu/velero => ../velero
