# Changelog

## Unreleased

**Bug Fixes**:

- Add payload type to occurrence payloads. ([#351](https://github.com/getsentry/vroom/pull/351))

**Internal**:

- Remove iOS legacy profile support. ([#296](https://github.com/getsentry/vroom/pull/296))
- Bump trufflesecurity/trufflehog from 3.60.1 to 3.60.2 ([#347](https://github.com/getsentry/vroom/pull/347))
- Bump trufflesecurity/trufflehog from 3.60.2 to 3.60.3 ([#348](https://github.com/getsentry/vroom/pull/348))
- Bump google.golang.org/grpc from 1.53.0 to 1.56.3 ([#349](https://github.com/getsentry/vroom/pull/349))
- Bump trufflesecurity/trufflehog from 3.60.3 to 3.60.4 ([#350](https://github.com/getsentry/vroom/pull/350))

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
