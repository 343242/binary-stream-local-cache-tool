# Offline Cache Simulator Implementation Plan

> For agentic workers: keep tasks modular and state-safe. When a task crosses CLI, generator, and engine boundaries, define the interface first and keep writes behind one narrow runner path.

**Goal:** Add an offline field-oriented workspace pressure tool that writes through the real storage engine into a specified empty root and produces real `meta/`, `wal/`, and `segments/` output.

**Architecture:** Introduce a new `cmd/cachesim` command over a small `internal/simtool` package. `simtool` owns profile selection, payload encoding, root validation, batch generation, execution, and final workspace reporting. The command layer stays thin.

**Tech Stack:** Go 1.22+, standard library only.

## Preconditions

- Do not modify `cachebench` semantics.
- Keep the simulator write path behind `cache.Open(...).WriteBatch(...)`.
- Reject non-empty target roots before opening the engine.
- Keep the first phase offline-only; no ticker loops or signal-driven long-running mode yet.

## Module Boundaries

- `cmd/cachesim/main.go`
  - flag parsing
  - process exit code handling
- `internal/simtool/profile.go`
  - profile names and resolved workload parameters
- `internal/simtool/encoder.go`
  - `SensorSample` definition
  - fixed-length payload encoding
- `internal/simtool/generator.go`
  - deterministic sample generation and batch slicing
- `internal/simtool/runner.go`
  - root validation
  - engine open/close
  - batch loop
  - report assembly
- `internal/simtool/report.go`
  - report type
  - text formatting
- `tests/unit/*`
  - profile, encoding, and root validation
- `tests/integration/*`
  - end-to-end simulator run against a temp workspace

## Task 1: Add failing profile and encoding tests

**Files:**
- Create `tests/unit/simtool_profile_test.go`
- Create `tests/unit/simtool_encoder_test.go`

- [ ] Write tests for:
  - default profile resolves to `medium`
  - explicit `large` resolves correctly
  - unknown profile returns error
  - encoded sample length is `24`
  - encoded bytes are stable for a fixed sample

- [ ] Run:
  - `rtk go test ./tests/unit/... -run 'TestSimProfile|TestEncodeSample' -v`

- [ ] Expect compile failure before implementation exists.

## Task 2: Implement profile and encoder modules

**Files:**
- Create `internal/simtool/profile.go`
- Create `internal/simtool/encoder.go`

- [ ] Add `ProfileName`, `Profile`, and `ResolveProfile(name string)` with:
  - default/empty => `medium`
  - `large` explicit only
- [ ] Add `SensorSample`
- [ ] Add `EncodeSample(sample SensorSample) []byte`
- [ ] Keep encoding fixed-width and deterministic

- [ ] Rerun focused unit tests until green.

## Task 3: Add failing root validation and report tests

**Files:**
- Create `tests/unit/simtool_runner_test.go`

- [ ] Add tests for:
  - nonexistent root is accepted
  - empty dir is accepted
  - existing file path is rejected
  - non-empty dir is rejected
  - report formatter includes root, profile, records, and segment count

- [ ] Run:
  - `rtk go test ./tests/unit/... -run 'TestValidateRoot|TestFormatSimReport' -v`

## Task 4: Implement runner/report skeleton

**Files:**
- Create `internal/simtool/runner.go`
- Create `internal/simtool/report.go`

- [ ] Add:
  - root validation helper
  - report struct
  - report text formatter
- [ ] Keep root validation independent from engine open so safety checks happen first

- [ ] Rerun focused unit tests until green.

## Task 5: Add failing generator tests

**Files:**
- Create `tests/unit/simtool_generator_test.go`

- [ ] Add tests for:
  - total sample count = gateways * points * rounds
  - timestamps advance by round
  - gateway and point ranges are deterministic
  - batch builder yields expected batch sizes for partial final batch

- [ ] Run:
  - `rtk go test ./tests/unit/... -run 'TestSampleGenerator|TestBuildBatches' -v`

## Task 6: Implement generator module

**Files:**
- Create `internal/simtool/generator.go`

- [ ] Implement deterministic iteration:
  - round
  - gateway
  - point
- [ ] Convert generated samples into `cache.RawRecord` batches
- [ ] Keep memory use bounded by constructing one batch at a time rather than the entire workload at once

- [ ] Rerun focused unit tests until green.

## Task 7: Add failing integration test for offline simulator run

**Files:**
- Create `tests/integration/cachesim_test.go`

- [ ] Add end-to-end test that:
  - runs the runner against a temp root
  - uses `medium` or a small override profile for test speed
  - verifies resulting workspace has `meta/`, `wal/`, `segments/`
  - verifies at least one `.seg` file exists
  - verifies report record count matches expected records

- [ ] Run:
  - `rtk go test ./tests/integration/... -run TestOfflineSimulatorCreatesWorkspace -v`

## Task 8: Implement execution loop and workspace stats

**Files:**
- Update `internal/simtool/runner.go`

- [ ] Implement:
  - profile-driven run config
  - `cache.Open(cache.DefaultConfig(root))`
  - batch loop with per-batch timing
  - engine close
  - post-run workspace stat collection
- [ ] Record:
  - total duration
  - average batch duration
  - max batch duration
  - segment count
  - WAL file count
  - workspace bytes
- [ ] Keep one write path only: runner -> engine -> WriteBatch

- [ ] Rerun focused integration tests until green.

## Task 9: Add CLI contract and command tests

**Files:**
- Create `cmd/cachesim/main.go`
- Create `tests/integration/cachesim_cli_test.go` if needed

- [ ] Add CLI flags:
  - `--root`
  - `--profile`
  - `--batch-size`
- [ ] Fail fast when `--root` is missing
- [ ] Print formatted report to stdout on success
- [ ] Return non-zero on validation/runtime failure

- [ ] Run focused verification for the command path.

## Task 10: Document and verify

**Files:**
- Update `README.md`
- Update `README_zh.md`

- [ ] Document:
  - `cachebench` vs `cachesim`
  - simulator root safety rules
  - medium/large profiles
  - example command lines

- [ ] Run full verification:
  - `rtk go test ./tests/unit/... -v`
  - `rtk go test ./tests/integration/... -v`
  - `rtk go test ./tests/benchmark/... -run TestBenchmarkContract -v`
  - `rtk go test ./...`

- [ ] If all pass, prepare a Lore-format commit covering spec, plan, code, tests, and docs.

## State-flow cautions

- Root validation must happen before `cache.Open`; do not let engine initialization mutate an unsafe path.
- Keep the simulator offline-only; do not add timers or wall-clock pacing into this first implementation.
- Do not accumulate the entire large profile payload set in memory at once.
- Do not add a second write path or bypass the engine.
- Do not let `cachesim` change the benchmark contract or output format of `cachebench`.
