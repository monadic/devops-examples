module github.com/monadic/devops-examples/cost-impact-monitor

go 1.21

require (
	github.com/google/uuid v1.6.0
	github.com/monadic/devops-sdk v0.1.0
	k8s.io/api v0.28.0
	k8s.io/apimachinery v0.28.0
	k8s.io/client-go v0.28.0
	k8s.io/metrics v0.28.0
)

replace github.com/monadic/devops-sdk => ../../devops-sdk