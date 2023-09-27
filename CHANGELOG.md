# Changelog

## Unreleased

**Features**

- Return the first and deepest issue detected. ([#317](https://github.com/getsentry/vroom/pull/317))

**Bug Fixes**

- Close remaining open events in Android profiles. ([#316](https://github.com/getsentry/vroom/pull/316))
- Enforce minimum frame duration for frame drop issue. ([#319](https://github.com/getsentry/vroom/pull/319))

**Internal**

- Rename the frame drop issue title. ([#315](https://github.com/getsentry/vroom/pull/315))
- Add new endpoint for regressed functions. ([#318](https://github.com/getsentry/vroom/pull/318))

## 23.9.1

**Internal**

- Bump the Go SDK to `v0.24.0`. ([#313](https://github.com/getsentry/vroom/pull/313))
- Remove the `TracesSampler` in favour of Inbound Data Filters. ([#313](https://github.com/getsentry/vroom/pull/313))

## 23.9.0

**Features**

- Improve frame drop detection algorithm. ([#304](https://github.com/getsentry/vroom/pull/304))
- Accept and return the time a profile started at in a timestamp field on Android. ([#306](https://github.com/getsentry/vroom/pull/306))
- Filter frame drop candidates based on thresholds. ([#308](https://github.com/getsentry/vroom/pull/308))

**Internal**

- Fix android issue frame detection. ([#305](https://github.com/getsentry/vroom/pull/305))
- Fix backward compatibility with Android profiles without timestamp. ([#307](https://github.com/getsentry/vroom/pull/307))
- Report all GCS errors again. ([#311](https://github.com/getsentry/vroom/pull/311))
- Use pipedreams to deploy. ([#312](https://github.com/getsentry/vroom/pull/312))

## 23.8.0

**Internal**

- Remove unwanted Println and add a CI rule to prevent further issues. ([#301](https://github.com/getsentry/vroom/pull/301))

## 23.7.2

**Internal**

- Change function fingerprints to uint32. ([#295](https://github.com/getsentry/vroom/pull/295))
- Send an occurrence on frozen frame drops. ([#297](https://github.com/getsentry/vroom/pull/297))
- Look forward instead of backwards when generating a call tree. ([#299](https://github.com/getsentry/vroom/pull/299))
- Use the instruction address to generate a frame's fingeprint. ([#298](https://github.com/getsentry/vroom/pull/298))

## 23.7.1

**Features**

- Add Android Regex symbols. ([#292](https://github.com/getsentry/vroom/pull/292))

**Internal**

- Remove Android method signature conversion. ([#294](https://github.com/getsentry/vroom/pull/294))

## 23.7.0

- No documented changes.

## 23.6.2

**Features**

- Release the Regex issue type detection. ([#286](https://github.com/getsentry/vroom/pull/286))
- Skip obfuscated frames from aggregation. ([#285](https://github.com/getsentry/vroom/pull/285)), ([#289](https://github.com/getsentry/vroom/pull/289))

**Internal**

- Run changelog CI action only on pull requests. ([#287](https://github.com/getsentry/vroom/pull/287))
- Enforce changelog modification. ([#282](https://github.com/getsentry/vroom/pull/282))
- Reintroduce a Cloud Build configuration. ([#288](https://github.com/getsentry/vroom/pull/288))

## 23.6.0

**Features**

- Add support for PHP application frame detection [(#186)](https://github.com/getsentry/vroom/pull/186)

**Bug Fixes**

- Fix node application frame detection [(#187)](https://github.com/getsentry/vroom/pull/187)

**Internal**

- Use environment variables for production config [(#185)](https://github.com/getsentry/vroom/pull/185)
