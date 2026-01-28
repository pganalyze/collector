# pganalyze-collector

![Version: 0.68.1](https://img.shields.io/badge/Version-0.68.1-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: v0.68.1](https://img.shields.io/badge/AppVersion-v0.68.1-informational?style=flat-square)

pganalyze statistics collector

**Homepage:** <https://pganalyze.com/>

## Source Code

* <https://github.com/pganalyze/collector>

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| affinity | object | `{}` |  |
| configMap.create | bool | `false` | Specifies whether a config map should be created. The config map can be used to set runtime environment variables |
| configMap.name | string | `""` | The name of the config map to load environment variables from. If not set and create is true, a name is generated using the fullname template |
| configMap.values | object | `{}` | Values to initialize the ConfigMap with. Only applicable if create is true |
| extraEnv | object | `{}` | Environment variables to be passed to the container Config settings can be defined here, or can be defined via configMap + secret |
| extraEnvRaw | list | `[]` | Environment variables to be passed to the container Config settings can be defined in raw form, for use with externally maintained env value sources (configMapKeyRef, fieldRef, resourceFieldRef, secretKeyRef) |
| fullnameOverride | string | `""` |  |
| image.pullPolicy | string | `"IfNotPresent"` | Overrides the image pull policy. |
| image.repository | string | `"quay.io/pganalyze/collector"` |  |
| image.tag | string | `""` | Overrides the image tag whose default is the chart appVersion. |
| imagePullSecrets | list | `[]` |  |
| nameOverride | string | `""` |  |
| nodeSelector | object | `{}` |  |
| podAnnotations | object | `{}` |  |
| podSecurityContext.runAsGroup | int | `1000` |  |
| podSecurityContext.runAsNonRoot | bool | `true` |  |
| podSecurityContext.runAsUser | int | `1000` |  |
| podSecurityContext.seccompProfile.type | string | `"RuntimeDefault"` |  |
| replicaCount | int | `1` |  |
| resources.limits.cpu | string | `"1000m"` |  |
| resources.limits.memory | string | `"1024Mi"` |  |
| resources.requests.cpu | string | `"1000m"` |  |
| resources.requests.memory | string | `"1024Mi"` |  |
| secret.create | bool | `false` | Specifies whether a secret should be created. The secret can be used to set sensitive runtime environment variables |
| secret.name | string | `""` | The name of the secret to load environment variables from. If not set and create is true, a name is generated using the fullname template |
| secret.values | object | `{}` | Values to initialize the Secret with. Only applicable if create is true |
| securityContext.allowPrivilegeEscalation | bool | `false` |  |
| securityContext.capabilities.drop[0] | string | `"ALL"` |  |
| securityContext.readOnlyRootFilesystem | bool | `true` |  |
| securityContext.runAsGroup | int | `1000` |  |
| securityContext.runAsNonRoot | bool | `true` |  |
| securityContext.runAsUser | int | `1000` |  |
| securityContext.seccompProfile.type | string | `"RuntimeDefault"` |  |
| service.create | bool | `false` | Specifies whether a service should be created for receiving logs via OpenTelemetry. This service is used when Postgres is running within the cluster and Postgres logs are sent out to the collector using log collectors like Fluent Bit |
| service.name | string | `"pganalyze-collector-otel-service"` | The name of the service to use. If not set and create is true, a name is generated using the fullname template. This is the name referenced by the log sender like Fluent Bit |
| service.port | int | `4318` | The port of service. This is the port referenced by the log sender like Fluent Bit |
| service.ports | list | `[{"name":"otel1","port":4318,"targetPort":4318},{"name":"otel2","port":4319,"targetPort":4319}]` | The list of port and target ports for OTEL logging. When this is specified, above port and targetPort will be ignored. If you need to have multiple log OTEL servers, use this. |
| service.targetPort | int | `4318` | The target port of the log OTEL server port. This should match to the port number specified with db_log_otel_server |
| service.type | string | `"ClusterIP"` | The type of service to create. |
| serviceAccount.annotations | object | `{}` | Annotations to add to the service account |
| serviceAccount.create | bool | `true` | Specifies whether a service account should be created |
| serviceAccount.name | string | `""` | The name of the service account to use. If not set and create is true, a name is generated using the fullname template |
| tolerations | list | `[]` |  |
| volumeMounts | list | `[{"mountPath":"/tmp","name":"scratch","subPath":"tmp"},{"mountPath":"/state","name":"scratch","subPath":"state"},{"mountPath":"/config","name":"scratch","subPath":"config"}]` | List of volume mounts to attach to the container |
| volumes | list | `[{"emptyDir":{},"name":"scratch"}]` | List of volumes to attach to the pod |

----------------------------------------------
Autogenerated from chart metadata using [helm-docs v1.14.2](https://github.com/norwoodj/helm-docs/releases/v1.14.2)
