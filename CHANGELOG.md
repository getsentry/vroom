# Changelog

## Unreleased

**Features**:

- Add support for speedscope rendering of Android reactnative profiles ([#386](https://github.com/getsentry/vroom/pull/386))
- Add callTree generation for reactnative (android+js) profiles ([#390](https://github.com/getsentry/vroom/pull/390))
- Use profiles that were not dynamically sampled to enhance slowest functions aggregation ([#300](https://github.com/getsentry/vroom/pull/300))

**Bug Fixes**:

- Label all node frames as system ([#392](https://github.com/getsentry/vroom/pull/392))
- Fix react-native (android) rendering issue ([#397](https://github.com/getsentry/vroom/pull/397))

**Internal**:

- Bump trufflesecurity/trufflehog from 3.63.4 to 3.63.5 ([#381](https://github.com/getsentry/vroom/pull/381))
- Bump trufflesecurity/trufflehog from 3.63.5 to 3.63.7 ([#385](https://github.com/getsentry/vroom/pull/385))
- Bump number of workers for flamegraph dynamically ([#388](https://github.com/getsentry/vroom/pull/388))
- Bump trufflesecurity/trufflehog from 3.63.8 to 3.63.9 ([#389](https://github.com/getsentry/vroom/pull/389))
- Bump trufflesecurity/trufflehog from 3.63.9 to 3.63.10 ([#391](https://github.com/getsentry/vroom/pull/391))
- Bump trufflesecurity/trufflehog from 3.63.10 to 3.63.11 ([#393](https://github.com/getsentry/vroom/pull/393))
- Ref(functions): Clean up Kafka message ([#394](https://github.com/getsentry/vroom/pull/394))
- Bump trufflesecurity/trufflehog from 3.63.11 to 3.64.0 ([#395](https://github.com/getsentry/vroom/pull/395))
- Bump trufflesecurity/trufflehog from 3.64.0 to 3.65.0 ([#396](https://github.com/getsentry/vroom/pull/396))
- Bump trufflesecurity/trufflehog from 3.65.0 to 3.66.1 ([#398](https://github.com/getsentry/vroom/pull/398))
- Bump trufflesecurity/trufflehog from 3.66.1 to 3.66.2 ([#399](https://github.com/getsentry/vroom/pull/399))
- Bump trufflesecurity/trufflehog from 3.66.2 to 3.66.3 ([#400](https://github.com/getsentry/vroom/pull/400))
- Bump trufflesecurity/trufflehog from 3.66.3 to 3.67.0 ([#401](https://github.com/getsentry/vroom/pull/401))
- Bump trufflesecurity/trufflehog from 3.67.0 to 3.67.1 ([#402](https://github.com/getsentry/vroom/pull/402))
- Refactor flamegraph workers sping-off logic ([#403](https://github.com/getsentry/vroom/pull/403))
- Bump trufflesecurity/trufflehog from 3.67.1 to 3.67.3 ([#404](https://github.com/getsentry/vroom/pull/404))
- Bump pre-commit/action from 3.0.0 to 3.0.1 ([#405](https://github.com/getsentry/vroom/pull/405))
- Bump trufflesecurity/trufflehog from 3.67.3 to 3.67.4 ([#406](https://github.com/getsentry/vroom/pull/406))
- Bump trufflesecurity/trufflehog from 3.67.4 to 3.67.5 ([#407](https://github.com/getsentry/vroom/pull/407))
- Bump golangci/golangci-lint-action from 3 to 4 ([#408](https://github.com/getsentry/vroom/pull/408))
- Remove experimental function regression issue ([#409](https://github.com/getsentry/vroom/pull/409))
- Bump trufflesecurity/trufflehog from 3.67.5 to 3.67.6 ([#411](https://github.com/getsentry/vroom/pull/411))

## 23.12.0

**Features**:

- Return the emitted regressions in response. ([#372](https://github.com/getsentry/vroom/pull/372))
- Support ingesting mixed android/js profiles for react-native ([#375](https://github.com/getsentry/vroom/pull/375)) --> this will let us store those profiles but it won't render the js part yet. A coming change will support that.

**Internal**:

- Bump google-github-actions/auth from 1 to 2 ([#371](https://github.com/getsentry/vroom/pull/371))
- Bump trufflesecurity/trufflehog from 3.63.1 to 3.63.2 ([#373](https://github.com/getsentry/vroom/pull/373))
- Bump actions/setup-go from 4 to 5 ([#374](https://github.com/getsentry/vroom/pull/374))
- Bump golang.org/x/crypto from 0.14.0 to 0.17.0 ([#380](https://github.com/getsentry/vroom/pull/380))

## 23.11.2

## 23.11.1

**Features**:

- Relicense under FSL-1.0-Apache-2.0. ([#366](https://github.com/getsentry/vroom/pull/366))

**Bug Fixes**:

**Internal**:

- Updated craft build rules to no longer bump version after move to FSL license
- Bump trufflesecurity/trufflehog from 3.62.1 to 3.63.0 ([#367](https://github.com/getsentry/vroom/pull/367))
- Bump actions/github-script from 7.0.0 to 7.0.1 ([#368](https://github.com/getsentry/vroom/pull/368))
- Bump trufflesecurity/trufflehog from 3.63.0 to 3.63.1 ([#369](https://github.com/getsentry/vroom/pull/369))
- Bump google-github-actions/auth from 1 to 2 ([#371](https://github.com/getsentry/vroom/pull/371))
- Bump trufflesecurity/trufflehog from 3.63.1 to 3.63.2 ([#373](https://github.com/getsentry/vroom/pull/373))
- Bump actions/setup-go from 4 to 5 ([#374](https://github.com/getsentry/vroom/pull/374))
- Bump github/codeql-action from 2 to 3 ([#377](https://github.com/getsentry/vroom/pull/377))
- Bump trufflesecurity/trufflehog from 3.63.2 to 3.63.4 ([#379](https://github.com/getsentry/vroom/pull/379))


## 23.11.0

**Features**:

- Support released key on regressed functions ([#355](https://github.com/getsentry/vroom/pull/355))
- Add frame-level platform field ([#364](https://github.com/getsentry/vroom/pull/364))

**Bug Fixes**:

- Add payload type to occurrence payloads. ([#351](https://github.com/getsentry/vroom/pull/351))
- Always classify sentry_sdk as system frame. ([#357](https://github.com/getsentry/vroom/pull/357))
- Update regression issue subtitle. ([#358](https://github.com/getsentry/vroom/pull/358))

**Internal**:

- Remove iOS legacy profile support. ([#296](https://github.com/getsentry/vroom/pull/296))
- Bump trufflesecurity/trufflehog from 3.60.1 to 3.60.2 ([#347](https://github.com/getsentry/vroom/pull/347))
- Bump trufflesecurity/trufflehog from 3.60.2 to 3.60.3 ([#348](https://github.com/getsentry/vroom/pull/348))
- Bump google.golang.org/grpc from 1.53.0 to 1.56.3 ([#349](https://github.com/getsentry/vroom/pull/349))
- Bump trufflesecurity/trufflehog from 3.60.3 to 3.60.4 ([#350](https://github.com/getsentry/vroom/pull/350))
- Add a pre-condition to check if the file already exists before write. ([#354](https://github.com/getsentry/vroom/pull/354), [#356](https://github.com/getsentry/vroom/pull/356))
- Bump trufflesecurity/trufflehog from 3.60.4 to 3.62.1 ([#353](https://github.com/getsentry/vroom/pull/353))
- Bump github.com/getsentry/sentry-go from 0.24.1 to 0.25.0 ([#359](https://github.com/getsentry/vroom/pull/359))
- Return a more appriopriate HTTP status code for duplicate profiles. ([#363](https://github.com/getsentry/vroom/pull/363))
- Bump actions/github-script from 6.4.1 to 7.0.0 ([#365](https://github.com/getsentry/vroom/pull/365))

## 23.10.1

**Features**:

- Update base Docker image to Debian 12. ([#340](https://github.com/getsentry/vroom/pull/340))

**Bug Fixes**:

- Turn non-monotonic samples wall-clock time into monotonic. ([#337](https://github.com/getsentry/vroom/pull/337))
- Classify browser extension paths as system frames. ([#344](https://github.com/getsentry/vroom/pull/344))

**Internal**:

- Bump golang.org/x/net from 0.8.0 to 0.17.0 ([#335](https://github.com/getsentry/vroom/pull/335))
- Move FixSamplesTime call to speedscope and calltrees methods([#339](https://github.com/getsentry/vroom/pull/339))
- Bump actions/checkout from 3 to 4. ([#341](https://github.com/getsentry/vroom/pull/341))
- Bump actions/github-script from 6.3.3 to 6.4.1 ([#342](https://github.com/getsentry/vroom/pull/342))
- Bump trufflesecurity/trufflehog from 3.16.4 to 3.60.1. ([#343](https://github.com/getsentry/vroom/pull/343))
- Bump actions/checkout from 3 to 4. ([#345](https://github.com/getsentry/vroom/pull/345))

## 23.10.0

**Features**:

- Return the first and deepest issue detected. ([#317](https://github.com/getsentry/vroom/pull/317))
- Release frame drop issue detection. ([#329](https://github.com/getsentry/vroom/pull/329))
- Append experimental to function regression issue. ([#334](https://github.com/getsentry/vroom/pull/334))

**Bug Fixes**:

- Close remaining open events in Android profiles. ([#316](https://github.com/getsentry/vroom/pull/316))
- Enforce minimum frame duration for frame drop issue. ([#319](https://github.com/getsentry/vroom/pull/319))
- Mark sentry frames as system frames when it's dynamically linked. ([#325](https://github.com/getsentry/vroom/pull/325))
- Do not return an occurrence for unknown function or when the stack is filled with them. ([#328](https://github.com/getsentry/vroom/pull/328))
- Add more Cocoa symbols for profiling issue detectors ([#336](https://github.com/getsentry/vroom/pull/336))

**Internal**:

- Rename the frame drop issue title. ([#315](https://github.com/getsentry/vroom/pull/315))
- Add new endpoint for regressed functions. ([#318](https://github.com/getsentry/vroom/pull/318))
- Return 502 from health endpoint on shutdown. ([#323](https://github.com/getsentry/vroom/pull/323)), ([#324](https://github.com/getsentry/vroom/pull/324))
- Health endpoint returns 200 instead of 204 on success. ([#326](https://github.com/getsentry/vroom/pull/326))
- Bump max profile duration for which we generate call trees. ([#330](https://github.com/getsentry/vroom/pull/330))

## 23.9.1

**Internal**:

- Bump the Go SDK to `v0.24.0`. ([#313](https://github.com/getsentry/vroom/pull/313))
- Remove the `TracesSampler` in favour of Inbound Data Filters. ([#313](https://github.com/getsentry/vroom/pull/313))

## 23.9.0

**Features**:

- Improve frame drop detection algorithm. ([#304](https://github.com/getsentry/vroom/pull/304))
- Accept and return the time a profile started at in a timestamp field on Android. ([#306](https://github.com/getsentry/vroom/pull/306))
- Filter frame drop candidates based on thresholds. ([#308](https://github.com/getsentry/vroom/pull/308))

**Internal**:

- Fix android issue frame detection. ([#305](https://github.com/getsentry/vroom/pull/305))
- Fix backward compatibility with Android profiles without timestamp. ([#307](https://github.com/getsentry/vroom/pull/307))
- Report all GCS errors again. ([#311](https://github.com/getsentry/vroom/pull/311))
- Use pipedreams to deploy. ([#312](https://github.com/getsentry/vroom/pull/312))

## 23.8.0

**Internal**:

- Remove unwanted Println and add a CI rule to prevent further issues. ([#301](https://github.com/getsentry/vroom/pull/301))

## 23.7.2

**Internal**:

- Change function fingerprints to uint32. ([#295](https://github.com/getsentry/vroom/pull/295))
- Send an occurrence on frozen frame drops. ([#297](https://github.com/getsentry/vroom/pull/297))
- Look forward instead of backwards when generating a call tree. ([#299](https://github.com/getsentry/vroom/pull/299))
- Use the instruction address to generate a frame's fingeprint. ([#298](https://github.com/getsentry/vroom/pull/298))

## 23.7.1

**Features**:

- Add Android Regex symbols. ([#292](https://github.com/getsentry/vroom/pull/292))

**Internal**:

- Remove Android method signature conversion. ([#294](https://github.com/getsentry/vroom/pull/294))

## 23.7.0

- No documented changes.

## 23.6.2

**Features**:

- Release the Regex issue type detection. ([#286](https://github.com/getsentry/vroom/pull/286))
- Skip obfuscated frames from aggregation. ([#285](https://github.com/getsentry/vroom/pull/285)), ([#289](https://github.com/getsentry/vroom/pull/289))

**Internal**:

- Run changelog CI action only on pull requests. ([#287](https://github.com/getsentry/vroom/pull/287))
- Enforce changelog modification. ([#282](https://github.com/getsentry/vroom/pull/282))
- Reintroduce a Cloud Build configuration. ([#288](https://github.com/getsentry/vroom/pull/288))

## 23.6.0

**Features**:

- Add support for PHP application frame detection [(#186)](https://github.com/getsentry/vroom/pull/186)

**Bug Fixes**:

- Fix node application frame detection [(#187)](https://github.com/getsentry/vroom/pull/187)

**Internal**:

- Use environment variables for production config [(#185)](https://github.com/getsentry/vroom/pull/185)
