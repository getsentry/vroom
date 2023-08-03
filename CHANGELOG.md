# Changelog

## Unreleased

**Internal**

- Change function fingerprints to uint32. ([#295](https://github.com/getsentry/vroom/pull/295))
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
