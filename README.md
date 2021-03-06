# conflint

`conflint` is an unified lint runners for various configuration files.

![image](https://user-images.githubusercontent.com/22009/85349934-b9504e80-b53a-11ea-9af4-1faa53a0d102.png)

It can run multiple lint runners in oneshot and output the result in a consistent and portable format so that
you can use it with e.g. [reviewdog](https://github.com/reviewdog/reviewdog) for surfacing the result as GitHub Pull Request reviews.

Compare vanilla output from the original tool and output from `conflint` to see how it works. It's really simple.

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

So, basically it runs various linters and aggregate results.

The small but important feature of it is to add line and colum numbers to every single lint error. This is achieved by assuming the beginning of every lint error message as a JSON Path-like notation.

`conflint` parses the path and searches for the YAML node at the path, and obtains the line and colum number to augument the output, so that the numbers can be used to annotate pull request diff line by line.

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

## GitHub Pull Request Check with conflint

This is possible by running `conflint` and `reviewdog` on GitHub Actions.

Use a workflow definition like the one below:

```
name: lint

on:
  pull_request:

jobs:
  lint:
    runs-on: ubuntu-latest
    container: mumoshu/conflint:latest
    env:
      REVIEWDOG_GITHUB_API_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    steps:
    - uses: actions/checkout@v1
    - name: conflint
      run: |
        set -vx
        export CONFLINT_LOG=DBEUG
        conflint run -efm "%f:%l:%c: %m" || true
        conflint run -efm "%f:%l:%c: %m" | reviewdog -efm="%f:%l:%c: %m" -reporter=github-pr-check -tee
```

See [gitops-demo](https://github.com/mumoshu/gitops-demo/blob/master/.github/workflows/lint.yml) repository for a working example, and [a check failure](https://github.com/mumoshu/gitops-demo/pull/2/files#diff-de00537bb5e8739d8c2bce941858ef79R8) reported by it.
