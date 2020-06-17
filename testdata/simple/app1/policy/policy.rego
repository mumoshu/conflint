package main

deprecated_deployment_version = [
  "extensions/v1beta1",
  "apps/v1beta1",
  "apps/v1beta2"
]

warn[msg] {
  input.kind == "Deployment"
  input.apiVersion == deprecated_deployment_version[i]
  msg = "apiVersion: Too old apiVersion. It must be apps/v1"
}

deny[msg] {
  input.kind == "Deployment"
  input.spec.template.spec.containers[_].securityContext.privileged == true
  msg = "spec.template.spec.containers[*]?(@.securityContext.privileged == true): `privileged: true` is forbidden"
}
