package processor

type metaLevel string

const (
	metaLevelRoot       metaLevel = "root"
	metaLevelTenant               = "tenant"
	metaLevelCloud                = "cloud"
	metaLevelGroup                = "group"
	metaLevelKubernetes           = "kubernetes"
)

var configFileExtensions = [2]string{"yml", "yaml"}

var restoreScript = `
#!/bin/sh -e

cd "$(dirname "$0")"

`
