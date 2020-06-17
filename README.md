# conflint

`conflint` is an unified lint runners for various configuration files.

It can run multiple lint runners in oneshot and output the result in a consistent and portable format so that
you can use it with e.g. [reviewdog](https://github.com/reviewdog/reviewdog) for surfacing the result as GitHub Pull Request reviews.

Vanilla `conftest`:

```console
$ conftest test app1/nginx.deploy.yaml -p app1/policy
WARN - app1/nginx.deploy.yaml - apiVersion: Too old apiVersion. It must be apps/v1
FAIL - app1/nginx.deploy.yaml - spec.template.spec.containers[*]?(@.securityContext.privileged == true): `privileged: true` is forbidden

2 tests, 0 passed, 1 warning, 1 failure
```

`conflint`:

```console
$ conflint run
app1/nginx.deploy.yaml:15:11: `privileged: true` is forbidden
Error: found 1 linter error
```

## Supported linters

- [conftest](https://github.com/open-policy-agent/conftest)
- [kubeval](https://github.com/instrumenta/kubeval)

## Integrations

- [reviewdog](https://github.com/reviewdog/reviewdog)

## Installation

- Pull [official docker images](https://hub.docker.com/repository/docker/mumoshu/conflint) containing conflint, conftest, kubeeval, and reviewdog binaries.
- Grab [release binaries](https://github.com/mumoshu/conflint/releases)

## Usage

`conflint run` runs linters as configured in your `conflint.yaml`. Include one or more configuration section(s) depending on which linter you want `conflint` to run.

### conftest

Any `conftest` policy message should start with a jsonpath expression for augmenting `conftest` errors with suspicious line and column numbers.

Example:

```
"spec.template.spec.containers[*]?(@.securityContext.privileged == true): `privileged: true` is forbidden"
```

Beyond that, all you need is providing `conflint` enough information about for which files and with which policy it should run `conftest`:

```yaml
conftest:
- files:
  - app1/*.yaml
  policy: app1/policy
```

In addition to the basic setup shown above, `conflint` covers most of conftest settings.

See `conftest run -h` and the below reference for more information:

```yaml
conftest:
- files:
  - app1/*.yaml
  policy: app1/policy
  # input type for given source, especially useful when using conftest with stdin, valid options are: [toml tf hcl hcl1 cue ini yml yaml json Dockerfile edn vcl xml]
  input: yaml
  # combine all given config files to be evaluated together
  combine: true
  # return a non-zero exit code if only warnings are found
  failOnWarn: true
  # A list of paths from which data for the rego policies will be recursively loaded
  data:
  - path/to/data
  # find deny and warn rules in all namespaces. If set, the flag "namespace" is ignored
  allNamespaces: true
  # namespace in which to find deny and warn rules (default [main])
  namespace:
  - foo
  - bar
```

### kubeval

Just provide target files in `conflint.yaml`:

```yaml
kubeval:
- files:
  - app1/*.yaml
```

In addition to the basic setup shown above, `conflint` supports wide range of kubeval options.

See `kubeval -h` and the reference conflint config for more information:

```yaml
kubeval:
- files:
  - app1/*.yaml
  # Disallow additional properties not in schema
  strict: true
  # Base URLs used to download schemas
  schemaLocations:
  - url/to/schema1
  - url/to/schema2
  # Skip validation for resource definitions without a schema
  # NOTE: This is a must-have when you use CRDs, as kubeval doesn't work against custom resources out-of-box
  ignoreMissingSchemas: true
  # A list of regular expressions specifying filenames to ignore
  ignoredFilenamePatterns:
  - some/regexp/pattern
  # A list of case-sensitive kinds to skip when validating against schemas
  skipKinds: true
```

## Reviewdog Integration

`conflint` formats every lint error message in `errorfmt`, so that using it with `reviewdog` is matter of running:

```
$ conflint run -efm "%f:%l:%c: %m" | reviewdog -efm="%f:%l:%c: %m"
```

To run reviewdog with conflint on GitHub Actions, use this snippet:

```yaml
- name: Run reviewdog
  env:
    REVIEWDOG_GITHUB_API_TOKEN: ${{ secrets.GITHUB_TOKEN }}
  run: |
    conflint run -efm "%f:%l:%c: %m" | reviewdog -efm="%f:%l:%c: %m" -reporter=github-pr-check
```

Please see [reviewdog's official documentation](https://github.com/reviewdog/reviewdog#option-2-install-reviewdog-github-apps) for how you can run it as a GitHub app.
