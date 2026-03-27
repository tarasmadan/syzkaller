# Automated Code Review Guidelines (DVYUKOV.md)

This file contains curated recommendations extracted from automated pull request reviews. Ingest this file into your LLM pipeline to ensure compliance with repository style and best practices!

## 1. HTML & User Interface

-   **Use `<details>` for Collapsibles:** Prefer the standard HTML `<details>` and `<summary>` tags over custom CSS "collapsible-hide" magic. Keep layouts lightweight and browser-native.
    -   *Example:* Use `<details><summary>Logs</summary>...</details>` instead of custom js/css state management.

## 2. Go Error Handling & Reliability

-   **Return Errors on Unexpected Conditions:** Do not swallow errors or return `nil` if something unexpected happens. If a condition is logically reached but invalid, return an error so the automated prompt or logs record it properly!
-   **Use `aflow.BadCallError` for Bad Inputs:** In the agent workflow (aflow tools), return `aflow.BadCallError` when inputs to a tool are invalid. This allows the LLM to understand its mistake and react dynamically. Let the LLM handle fixing paths, checking if files exist, or fixing hex encodings.

## 3. Testing Best Practices

-   **Prefer `testify/require` Assertion Suite:** Use `require.NoError(t, err)` from `github.com/stretchr/testify/require` instead of doing manual `if err != nil { t.Fatal(err) }` checks. This is the repository standard.
-   **Use Raw String Literals for Multi-Line Strings:** When specifying multi-line test data or expected outputs, use raw string literals. Specifying things with manual `\n\n\n` degrades readability.

## 4. Code Cleanup

-   **Remove Unused Types/Structs:** If an endpoint or action changes and a struct or field becomes unused, remove it completely. Do not leave placeholder types or dead code.
-   **Limit Dead Logs:** Avoid using `log.Printf` inside core libraries if there are no downstream consumers reading them. Log only if it is surfaced to workflows or tools that need the logging output.

## 5. Concurrency & Races

-   **Test with `-race` Binaries:** If you touch shared state (e.g. `Choices` or global maps), run your tests with `-race` binaries to surface data races. Data races lead to flakes and undefined behavior.

## 6. Resource Leaks

-   **Process Waits and Pipes:** Always call `Wait()` on launched commands (`os/exec.Command`). Leaving child processes zombies leaks OS resources!
-   **File Descriptors and ULIMITS:** The repository sets ulimit 255 for open file descriptors. Leaking even a single fd per test will crash the workflow. Use `defer` to close file pipes and descriptors.
-   **Goroutine Leaks:** In tests using background concurrency, use `checkGoroutineLeaks()` to ensure tests don't leave background workers alive.

## 7. Crashing vs Silencing Invariants

-   **Prefer Crashing Loudly for Internal Invariants:** If reaching a code path means a critical internal logic corruption (e.g., UI state page without description, or unreachably corrupted in-memory cache), prefer crashing loudly with a stack trace. Return an error for inputs, but crash for broken engine invariants! Suppressing it silences the cause and makes it impossible to debug.


## 8. Go Coding Style & Structure

-   **Line Column Length (Go):** Cap Go lines at **120 columns**. Do not format using 80 columns (no punch card constraints).
-   **Indentation Tabs vs Spaces:** Go uses tabs. Never replace tabs with spaces.
-   **Combine Var Declaration and Initialization:** Combine `var x = foo()` to `x := foo()` where possible. We do not use C89 start-of-block declarations.
-   **Unexport Non-Exposed Symbols:** If a symbol is only used in a single file, unexport it (`lowerCaseName`).
-   **Unused Code:** If a struct or field becomes unused after a refactor, remove it completely. Do not leave placeholder types.

## 9. Advanced Errors & Interface Deserialization

-   **Errors on Mistyped Args:** Always validate inputs and return a hard error rather than failing silently if an arg is mistyped. Silent failures mask bugs.
-   **Strict Mode Deserialization Gotchas:** Be careful with strict deserialization of test programs. If syscalls or arguments change across syzkaller revisions, strictDeserialization will fail. 

## 10. Modern Tooling Standards

-   **New CLI Helper Tools Initialization:** Standardize on using `tool.Init()` for newer tools. It configures default logging and crash reporting uniformly.


## 11. Zero Initialization (Go)

-   **Rely on Zero Values Defaults:** Do not manually initialize variables to their zero values (e.g. `x = 0`, `y = ""`, or `m = map[type]type{}`). Rely on standard Go defaults!

## 12. Constructor Naming (Go)

-   **Use `NewFoo` or `MakeFoo` for Constructors:** In Go, constructors are standardly named `NewFoo` or `MakeFoo`. If methods are defined on a pointer `*Foo`, return a pointer from the constructor!

## 13. Avoid Tunables Bias

-   **Do Not Expose Unnecessary Boolean/Int Tunables:** Deciding defaults is the hardest part. If you cannot decide, nobody can decide. Choosing a thousand tunables makes the tool unrunnable for standard users. Pick one setting!

## 14. Documenting Tests with Reports

-   **Each Real Pattern Needs a Test Report:** When adding parsers or crash reports, include sample test reports in `pkg/report/testdata/...`. This is super useful for regression testing.

## 15. No C89 Start-of-Block Declarations (C)

-   **Combine Variable Declarations & Initialization:** Do not use old C89 start-of-block declarations in C++ sources. Combine them where possible.

## 16. OS Abstractions in Executor (C)

-   **Keep `executor.cc` Free of `#ifdefs`:** Avoid OS-specific `#ifdefs` inside `executor.cc`. Abstract differences using OS-specific header files (e.g. `executor_bsd.h`). Use pointers and structs to hide differences!


## 17. Complexity O(N) vs O(N^2)

-   **Use Map Lookups for Filtering Excludes:** Double loops for excludes are slow (O(N^2)). Put exclude rules into a map first, then do a single pass over items to remove them. High performance matters in the manager!

## 18. Idiomatic Go Map Existence Check

-   **Prefer `if !keys[entry]` for Boolean Maps:** Do not use `if _, ok := keys[entry]; !ok` if the map values are booleans. Just ask for the boolean value!
    -   *Example:* `if !keys[entry] { ... }` is more concise.

## 19. Tests for Utility Functions

-   **Unit-Test Utility Functions:** Any pure data transformation or utility function (e.g., regex filtering groups) should have its own unit tests. Testability is critical.

## 20. Pure Data Architectures

-   **Keep Analytics/Verification Pure Data Processing:** A program comparator should only accept a `*Prog` and `N` results and produce a verdict. It should not know about manager configs, VM stop channels, or binary locations.

## 21. Scoping Flags in `main()`

-   **Declare CLI Flags in `main` Function:** Global variables are still evil. Scoping flags in `main` allows for better testing and prevents polluting the global namespace.

## 22. Avoid Visual Noise Prefixes

-   **Do Not Prefix Fields with "syz":** All structs and fields in this repo are syz-something. Prefixing `syz_exec_total` over `exec_total` adds visual noise.


## 23. Simplify Complex Tracking Mechanisms

-   **Favor Simpler State Tracking:** Do not use double maps or slices (e.g. `res` and `left` maps) where a single counter or single ordered slice can do it. Reducing state variables makes logic easier to read and less error-prone.

## 24. Arch Struct Fields over Switches

-   **Use Arch Struct Fields:** Instead of logic-driven switches (e.g., `switch arch { case AMD64: ... }`), add fields to your `Arch` structs. Keep low-level arch details in the arch definitions.

## 25. Pass Objects / Pointers Early

-   **Pass Pointers over IDs Where Applicable:** If you have an index `progIdx` inside a function, map it to `progInfo` as early as possible and pass the pointer around. It avoids repeating lookups and shortens the code context.

## 26. Cross-Platform Filename Handling

-   **Use Raw String Literals for Regexes and `filepath.Join` for Paths:** Do not use standard strings for regexes (taps `"` vs ticks ``). Do not use hardcoded `/` characters in paths. Use `filepath.Join`.

## 27. Avoid Garbage Scrambles on Large Data

-   **Avoid Casting Large Buffers to Strings:** Use `FindSubmatch` or byte processing on large files instead of a quick `string` cast if the string isn't used. Generating garbage slows down the manager.

## 28. Tests Are Sacred: Do Not Just Disable Them

-   **Fix Failing Tests instead of Disabling:** Disabling a test removes coverage. If a test fails, figure out *why* (e.g., mismatched PCs vs coverage callbacks) and fix it. If disabling is necessary, add an explanatory comment!


## 29. All Persistent Formats Must Be Deterministic

-   **Deterministic Serialization:** Map iteration order is non-deterministic. Always sort keys or specify deterministic output when serializing to any persistent format (config, logs). Random order breaks tests and is confusing.

## 30. No Dead Code (YAGNI)

-   **Do Not Add Code "Just In Case":** It's hard to predict what will be needed and in what form. Dead code breaks tests and clutters the workspace. If you don't use it now, remove it!

## 31. Meaningful Temp File Names

-   **Use Recognizable Temp File Prefixes:** When creating temp files in `/tmp`, use a prefix like `syz-` to distinguish them from other system files. It makes it easier to clean up if a job crashes.

## 32. Normal Functions Over Closures

-   **Prefer Free/Normal Functions over Closures:** If a closure doesn't capture anything from its outer context, it's run-time overhead and deepens nesting. Refactor it to a standard top-level function.

## 33. Avoid Visual Redundancy in Method Names

-   **Don't Repeat Type Names in Method Names:** If a method is on the `Linux` type in `linux.go`, naming it `fooLinux()` is redundant. Keep it `foo()`.

## 34. Avoid Unnecessary String Casts for Formatting

-   **Let `%s` Handle Struct String Conversions:** You don't need to do `v.String()` if you pass `v` to a `%s` format specifier in `fmt.Printf`. Passing the object directly is idiomatic.


## 35. Positive Entity Naming over Negation

-   **Favor Positive Names:** Do not use negations in variables where simple positive entities can describe the state. For example, use `active` instead of `not_inactive`. Positives are easier to reason about.

## 36. API & Config Conservatism (YAGNI)

-   **Do Not Introduce Public Config Parameters "Just In case":** Users will use them in complex ways (e.g. ratios) that restrict your ability to modify implementations dynamically in future. If you don't need it for specific experiments, do not export it.

## 37. Extract Loop Defer Targets

-   **Extract Loop Bodies Using Defers:** If a loop body creates a resource that needs `defer Close()`, extract the loop body into a separate function so the defer fires *per iteration* instead of at the end of the outer function.

## 38. Consolidate Logging & Startup Messages

-   **Do Not Mix Multiple Logger Libraries:** Use the internal `log` package consistently. For startup errors where date/time tags are distracting, use `tool.Failf` instead of `log.Fatalf`.

## 39. Count Backwards for Simplification

-   **Count Down When Scanning:** Counting down `asyncLeft := 7 ... if asyncLeft > 0 { asyncLeft-- }` is often shorter and simpler than tracking `asyncCount` and `maxAsync`.

## 40. Recovery Tests for Parsers

-   **Add Tolerance and Recovery Tests:** Parsers should be tested not just for valid/invalid cases, but for recovery from invalid cases (e.g., skip an unknown property and continue parsing the next valid property).


## 41. Prefer `openat` Over `open` in Syscall Descriptions

-   **Use Modern Syscalls:** Standard `open` syscall is not present on many newer architectures (like `arm64` or `riscv64`). Use `openat` with `AT_FDCWD` instead of standard `open` in `sys/*/*.txt` definitions.

## 42. Scope Resource and Syscall Names

-   **Avoid Generic Names:** Do not name a resource `fd_attrs` or a syscall `write$attrs`. Use descriptive prefixes like `fd_damon_attrs` to avoid collisions with other subsystems.

## 43. Uniformity Helps Testing

-   **Keep Logic Uniform Across Modes:** If you have multiple execution modes (e.g. fork-server vs no fork-server), keep the logic flow as uniform as possible between them. Testing one mode (the default) much better than the other is common, so uniformity prevents bugs in the less-tested mode.

## 44. Inherited Resources Get Standard Syscalls Automatically

-   **Do Not Redefine `read` and `close`:** If a resource inherits from `fd`, the fuzzer automatically knows how to `read` and `close` it. You do not need to redefine these in your subsystem text files unless you need to pass specific data.

## 45. Errors Early, No Nesting (Idiomatic Go)

-   **Check Errors Early:** Go functions should return errors early (standard `if err != nil` check) rather than trapping the actual success logic inside a large `else` block or deeply nested conditions.

## 46. Multi-Line Differential Helpers in Tests

-   **Use Line-by-Line Diffs for Tests:** If you compare large multi-line text outputs (e.g. JSON or serialization logs), use test helpers that print usable per-line diffs (like `c.expectEQ` in standard syzkaller tools) instead of manual string comparisons.


## 47. Condition Variables Must Use a Loop

-   **Wait Predicates:** Condition variables must always be check in a loop to prevent race hangs if the predicate was already true before wait:
    ```go
    for !waitPredicate {
        cond.Wait()
    }
    ```

## 48. Avoid Infinite Goroutine Spawning in Pools

-   **Track Wait States:** If a call to `generate()` doesn't immediately increase the queue length checked by the pool, don't spawn goroutines in an untracked `for` loop. It can spawn infinitely many before the first one finishes.

## 49. Event-Driven over Polling/Sleeps

-   **Red Flags for Sleeps:** A loop with a sleep is always a red flag. Design when `select` can wait for the event it needs to handle instead of periodic polling.

## 50. Assign Unique IDs for Task Submissions

-   **Trace Race Flakes:** Assign a unique ID to every submission of a task to a runner (or retry) to distinguish between results of previous stale runs and current runs.

## 51. Sort for Determinism in Code Coverage

-   **Sort Map Keys:** Map iteration order is non-deterministic. If you generate coverage profiles, random fluctuations split across PRs can break CI. Always sort keys for determinism.

## 52. Log Hygiene (Actionable Logs only)

-   **Do Not Produce Thousands of Unactionable Verbosity 0 Logs:** If an error occurs frequently but is expected/tolerated, do not log it at verbosity 0. It obfuscates real problems and floods the console.


## 53. Use Functional Callbacks for Shared Setup/Teardown

-   **Callback Patterns for Complex Scopes:** To share complex mounting/unmounting logic (e.g., between Linux and Android setups), pass a callback function to the shared setup function:
    ```go
    func mountDir(img string, cb func(dir string) error) error {
        // ... mount ...
        err := cb(tempDir)
        // ... unmount ...
        return err
    }
    ```

## 54. Use `go:embed` for HTML Templates

-   **Native Embedding:** Use standard Go embed tags (`//go:embed`) to bundle HTML or text templates instead of inlining them in Go strings.

## 55. Use Exact Kernel Field Names in Descriptions

-   **Preserve Naming in `sys/*/*.txt`:** Always use the actual kernel names for fields in system call descriptions. Changing them breaks searchability for kernel developers and static analysis tools.

## 56. Use `const[0]` for Reserved Fields in Descriptions

-   **Optimize Mutations:** If a field is unused or reserved by the kernel, use `const[0]` (with correct type size) instead of an open integer type so that the fuzzer does not waste randomness mutating it.

## 57. Remove Unused Optional Features

-   **Don't Ignore Optional Flags Silently:** If a feature/flag becomes unused or superseded, delete it entirely rather than preserving it as an empty or ignored stub.

## 58. Avoid Naked Returns and Named Return Overuse

-   **Standard Return Syntax:** Naked returns (`return` without values when named returns are declared) should be avoided unless they significantly simplify repetitive code. They are generally not idiomatic in Go.


## 59. Format String Safety for Logs

-   **Use String Explantions in Logf:** Dynamic outputs like `out` may contain `%` commands. To prevent faults or format errors, use `Logf(1, "%s", out)` instead of `Logf(1, out)`.

## 60. Match Scope Operations (Mount/Unmount)

-   **Symmetrical Cleanup:** If you mount an image in a loop (or create a resource), ensure you unmount/clean it up symmetrically. If mounting can happen multiple times, track counts or unmount twice.

## 61. Synchronous Code over Immediate Channel Receives

-   **Simplify Immediate Blocking:** There is no point in starting a goroutine and immediately blocking on its response channel (e.g. `go func() { ... }() \n <-errc`). That just adds overhead and complexity. Inline it synchronously.

## 62. Unexport Unused Package Symbols

-   **Clean Package APIs:** If a symbol is only used within its package, keep it unexported. Do not expose public APIs "just in case".

## 63. Tests for Template Changes

-   **Verify Template Renders:** When modifying report or email templates, add at least one test verifying the rendering output. Broken templates crash at runtime and are hard to debug without tests.


## 64. Boundary Validation for Persistent Schema

-   **Validate Inputs at Datastore Boundaries:** Incoming data should be validated before it is written to persistent datastores to prevent corruption or invalid states.

## 65. Store Enum Codes over Descriptive Strings

-   **Database Schema Optimization:** In datastore entities, prefer storing enum/integer codes instead of full descriptive strings. Strings are slower, take more space, and are harder to query. Convert them to strings at the presentation layer.

## 66. Mark Pseudocall Responses with `ignore_return`

-   **Feedback Invariance:** If a syscall/pseudocall returns a random value (e.g., `zx_clock_get` time or random system IDs), mark the definition with `ignore_return` in `sys/*/*.txt`. This prevents Syzkaller's fallback coverage feed from treating random values as new coverage signal.

## 67. Use `procid` for Sandboxed Scopes

-   **Reliable Resource Cleanup:** Instead of using the process's real PID (host-assigned), use the `procid` variable (mapped between 0 and `procs`). This ensures resources (like temporary network namespaces `staging_ns.%d`) are reused and cleaned up properly.

## 68. Fail Fast on Feature Degradations

-   **Loud Failures on Proved Features:** If a host feature check (e.g., in `pkg/host`) says a feature is supported, do not silently ignore runtime errors for that feature in the executor. Use a `fail()` check to intercept regressions early.

## 69. Go Package Naming Convention

-   **Lower Case, No MixedCaps:** Verify Go package names are all lowercase without underscores or mixed caps (e.g., `pkg/fuzzer`, `vm/qemu`). If ambiguity is possible, prefer slightly longer, more descriptive names that are unique (e.g., `vm/proxyapp/proxyrpc`).


## 70. Positive Boolean Predicates

-   **Prefer Positive Logic:** Positive conditions are usually easier to understand and verify than defining a predicate as a negation of multiple negative conditions.

## 71. Batch Shell Commands for Efficiency

-   **Reduce Connection Overhead:** Combine multiple shell commands into a single invocation (e.g., `adb shell 'cmd1; cmd2; cmd3'`) instead of establishing multiple separate SSH or ADB connections. This is faster and easier to read.

## 72. Use Standard `Close()` Naming for Cleanup

-   **Idiomatic Go Cleanup:** Follow standard Go conventions for resource cleanup methods. Use `Close()` instead of `Free()` or `Teardown()` for standard resources and objects.

## 73. Use Loop Post-Statements for Retry Sleeps

-   **Prevent Forgotten `Sleep` on `continue`:** Structure retry loops so the sleep is the post-statement to ensure it is executed even if the body invokes `continue`:
    ```go
    for attempt := 0; attempt < max; attempt++, time.Sleep(delay) {
        // ...
        if err != nil {
            continue
        }
    }
    ```

## 74. Loop Index Naming Conventions

-   **`i` is for Index:** Use the single-letter variable `i` for loop indices. Use descriptive names (e.g., `el`, `obj`, `itm`) for the slice elements themselves.

## 75. Do Not Wrap Errors Without Context

-   **Hygiene in Error Passing:** Standard Syzkaller library functions already prefix errors with basic context. Do not wrap an error with a generic message (e.g., `fmt.Errorf("failed to query: %w", err)`) unless you are adding specific, helpful data (such as the target object or ID).


## 76. Differentiate Plugin Bugs from Kernel Bugs with `SYZFAIL`

-   **Fault Location Hygiene:** When a VM or ProxyApp plugin fails due to its own internal bugs, write the error message prefixed with `SYZFAIL: <context>`. If you output a plain error, Syzkaller may classify it as "lost connection to test machine" (implying a target kernel crash), which pollutes reports.

## 77. Use Closures to Declare Local Utility Functions

-   **Deduplicate Scoped Logic:** If a block of logic is repeated multiple times inside a single function (e.g., list flushes or error cleans), declare it as a closure at the top. This keeps the function body readable and prevents drift between duplicates.

## 78. Performance: Prefer Direct Slicing over Readers

-   **Memory Efficiency:** When parsing chunks of byte data, prefer direct slicing `chunk, data = data[:n], data[n:]` instead of using complex `io.Reader` subslicing which generates garbage.

## 79. Localize Complexity to Where Needed (Mutation)

-   **Minimalist Interface Pollution:** Do not pollute base types (like `prog.Arg`) with specific getters/setters for rare features (like data compression). Implement compression decompression inside the mutation logic where it is actually used.

## 80. Fail Fast on Go Init Breakage

-   **Loud Regressions:** Use `panic()` inside Go init or description-parsing tools to ensure that parsing breakage is caught during tests rather than slipping into production.


## 81. Drop Default Values in Ast Declarations

-   **Code Hygiene:** If an AST type field defaults to `0` or `false`, do not include it in the declaration. Keep the declaration as minimal as possible.

## 82. Add Positive Tests for New Compiler Features

-   **Test Coverage:** When adding a new feature or constraint, add a negative test to `errors.txt` (or similar error-catching testdata) and a positive test to `all.txt` (or equal functional testdata) to document how it *should* work.

## 83. Minimize Expensive Type Conversion Checks

-   **Performance Optimization:** When enter a loop to parse arguments or buffers, restrict type-cast and memory allocation checks (such as `[]byte` to `string`) to only those types that actually need it (such as `BufferString` or `BufferFilename`). Do not run them for generic types like `BufferBlob`.

## 84. Skip Complex Mutations for Feature Verification

-   **Prudent Logic Addition:** If a feature (such as hints generation for compressed buffers) is complex and its utility is unproven, skip its implementation for now and add a comment explaining why.

## 85. Panic on Impossible In-Memory Faults

-   **No Silent Ignoring or Untested Returns:** For in-memory operations that cannot fail under normal conditions (such as zlib compression of a local byte slice), make the function panic internally rather than returning a useless error that callers will ignore.

## 86. Validate Invariants Early (Deserialization)

-   **Fail Fast on Corruption:** For complex arguments (such as `BufferCompressed`), validate that they can be decompressed during program deserialization, rather than letting a corrupt argument fail quietly later during execution or mutation.

## 87. Use `AUTO` for Auto-Computing Constants in Tests

-   **Automated Lengths and Offsets:** Use the `AUTO` keyword in system call description tests so that the framework computes correct offsets and lengths automatically.

## 88. Guard Against Underflow in Length Checks

-   **Runtime Safety:** When checking for a minimum header size (such as reading from a buffer), do a strict check `if (len < HEADER_SIZE) return -12; len -= HEADER_SIZE` to prevent negative underflow in length calculations.

## 89. Log Invalid User Requests as info, Not Errors

-   **Hygiene in Log Escalation:** If an error is caused by bad user input (e.g., `BadTestRequestError`), do not log it as `log.Errorf` as it pollutes administrative error dashboards. Use info logs or specific error types for validation errors.

## 90. Standard Copyright Header for SysCall Text Desc

-   **Text Descriptions Hygiene:** Ensure all new `.txt` system call descriptions carry the standard open source copyright header for the Syzkaller project.


## 91. Use `nil` Channels to Disable Select Cases

-   **Go Concurrency Idiom:** Instead of creating a dummy channel that never fires, use a `nil` channel in a `select` statement. A read from `nil` channel blocks forever, effectively disabling that case without wasting resources.

## 92. Prefer Explicit Expression Over Overwrites

-   **Code Cleanliness:** When initializing an object with various state options (e.g., standard vs parallel workers), write explicit `if/else` branches for all variations. Do not initialize common fields and then let another branch overwrite them, as it confuses readers.

## 93. Place Executable Utilities in the `tools/` Directory

-   **Workspace Organization:** All standalone binaries, CLI utilites, and script entry points should be stored in the top-level `tools/` directory rather than nested deep in package library paths.

## 94. Standardize CLI Error Reporting via `pkg/tool`

-   **Tooling Consistency:** Use the `tool.Init` and `tool.Failf` utilities from `pkg/tool` for clean standard error tracking and formatting.

## 95. Use Table-Driven Tests with `t.Run`

-   **Test Isolation:** Isolate subtests within table-driven tests using `t.Run("test-name", func(t *testing.T) { ... })`. This allows running individual tests and keeps log output clean for failing ones.

## 96. Use `runtime.NumCPU()` for Default Worker Counts

-   **Simplicity over Tunables:** Do not introduce configuration flags for worker threads if they can be defaulted to `runtime.NumCPU()`. Avoid exposing knobs unless absolutely necessary.

## 97. Exclude Intermediate/Editor Files in Scans

-   **File Scanning Hygiene:** When scanning the tree for source files, ensure you filter out binary objects (`*.ko`, `*.o`) and editor swap files, or use a strict allowlist of source extensions.

## 98. Use Standard Auto-Generated Captions for Go Code

-   **Machine Generated Files:** Use the standard Go caption format `// Code generated ... DO NOT EDIT.` so it is recognized correctly by all Go linting and formatting tools.


## 99. Compile Regexps Early with `MustCompile` for Tested Data

-   **Tested Invariants:** If a regex is built from trusted static sources (such as a checked-in configuration or parsed during `init`), use `MustCompile` instead of propagating execution errors. If there's a typo, tests will catch it early.

## 100. Fail Fast on Subsystem Name Validation Errors

-   **Invariant Consistency:** If a subsystem name validation fails (e.g., character violations, length limits), return an error immediately. Do not allow empty or broken names to propagate silently to the dashboard.

## 101. Avoid Brittle Command-Line Flag Validation in `main()`

-   **Tooling Hygiene:** Rely on standard flag parsing instead of rigid custom checks in `main()` that might conflict with automatic flag defaults (such as `-help` usage printing).

## 102. Ensure Correct Double Calling for `defer tool.Init()()`

-   **Cleanup Functions:** When using `tool.Init()`, ensure you write `defer tool.Init()()` as it returns a cleanup function that must run at function exit.

## 103. Prioritize Large Coherent Packages Over Tiny Granularity

-   **Go Idioms (Cheney Rule):** Favor fewer, larger packages over micro-packages. Combine tight coupling (such as `match` logic and `subsystem` data structures) into single packages to reduce interface pollution.

## 104. Minimize Transactions for Read-Heavy Queries

-   **Performance Optimization:** Pull reading heavy queries (such as scanning previous crashes) *out* of database write transactions to prevent contention and timeouts.

## 105. Throttle Automatic Metric Updates

-   **Load Management:** Do not recalculate complex metrics (such as subsystem associations) on *every* single crash run. Limit updates to the first report or the first reproduction to prevent flip-flopping state and DB load.

## 106. Use `omitempty` for Lean JSON Outputs

-   **JSON Hygiene:** Use `json:",omitempty"` for fields where empty values (`""` or `0`) are normal, keeping the exported telemetry lean.


## 107. Add Tests for Complex Index Lookups (e.g., `sort.Find`)

-   **Test Boundary Conditions:** When using binary search or offset lookups, always write tests for boundary cases (such as value < first or > last item).

## 108. Keep One-Liner Initializations Inline in Tables

-   **Code Size Hygiene:** Inlining simple table configurations (such as sanitizer disable functions) in tables helps keep code size small and prevents jumping between locations in source to read one-liner functions.

## 109. Log Human-Readable Messages for Skipped Bisection Commits

-   **Debugging Hygiene:** Ensure the bisection log explains *why* a commit is ignored (e.g., `Skip: Require at least 50% successful runs to say it is a good commit`).

## 110. Write Binary Formats Manually over Relying on Target Tools (Executor)

-   **Self-Containment:** If a target VM image might not have a utility (such as `mkswap`), write the format manually using file writes if it's simple (e.g., magic `"SWAPSPACE2"` write).

## 111. Avoid Printing Errno in `failmsg` If the Utility Does It

-   **Executor Logging:** Do not double-print errors where standard utilities take care of it automatically.

## 112. Do Not Export Raw State Hierarchies Directly in Public JSON APIs

-   **API Stability:** Abstract datastore hierarchies for public JSON APIs to keep them stable. Exposing raw internal hierarchies ties the API too tightly to internal implementation.

## 113. Prefer Specific Descriptive Package Names over Generic Ones

-   **Go Naming Idioms:** Always prefer specific names like `minimize` over `generic`. Package names define what calling code looks like (`minimize.Slice` vs `generic.Slice`).

## 114. Pull Long Closures Out-of-Line for Config Readability

-   **Code Hygiene:** Move large closure definitions out of declarative configuration builders so the builder remains easy to read.


## 115. Embed Configuration Structs to Reduce Code Clutter

-   **Go Idioms:** Use unnamed fields for configuration structs inside your execution contexts or state wrappers to reduce repetitive `.cfg.` or `.config.` accesses.

## 116. Keep "UNKNOWN" Fallback Names for Reporting Types

-   **Reporting Hygiene:** Do not use empty strings or undefined labels for reporting types if they might be displayed to the user. State them explicitly as `"UNKNOWN"`.

## 117. Add Fast-Path Checks for Zero Modules in Canonicalizers

-   **Performance Optimization:** If an operation assumes modules may load, add a fast-path return `if len(modules) == 0` to prevent unnecessary iterations on standard kernels.

## 118. Wrap CLI Email Documentation at 78-80 Columns

-   **Documentation Hygiene:** Ensure plain text emails or documentation examples fit standard 78-80 column limits for terminal-based email clients.

## 119. Track Message Links/IDs over Authors for Disputed Tags

-   **Dashboard Hygiene:** Tracking the specific email link or message ID that auto-labeled a bug is often more useful for disputes than the author's email alone.

## 120. Check for Short Writes in Output Loops (Executor)

-   **Executor Hygiene:** When writing data buffers to files, do not just check if the return value is `< 0`. Also check if it is `< sizeof(buffer)` to catch short writes.

## 121. Drop Unnecessary `else` After `return` or `break`

-   **Go Style (Effective Go):** Remove `else` blocks when the previous branch unconditionally returns or breaks. Keeps indentation lean.


## 122. Use `osutil.IsExist` Instead of Home-Grown Presence Checks

-   **Uniformity:** Use `osutil.IsExist` for existence checking rather than writing custom `os.Stat` calls where possible.

## 123. Check `match != nil` Rather than `len(match)` for Static Group Matches

-   **Regexp Idioms:** When using regexps with capture groups, `len(match)` is always static if `match` is not nil. Check for standard `match != nil` instead.

## 124. Use `MustCompile` for Static Built-In Regexps to Panic on Failure

-   **Panic on Impossible:** Since built-in paths or hardcoded regexps cannot fail at runtime, use `MustCompile` which will panic early if flawed. Don't silently ignore or return errors.

## 125. Use Channels for Semaphores if Weights Are Not Needed

-   **Concurrency Hygiene:** A standard send/receive channel is simpler for basic gating than importing complex weighted semaphores.

## 126. Panic If Standard Telemetry Handlers Setup Fails in `main`

-   **Fail Fast:** If listening for pprof metrics fails to bind, panic immediately. Telescience is useless if it's broken silently.

## 127. Move Common Telemetry Settings to Standard Packages (like `vmimpl`)

-   **Uniformity:** Avoid hard-coding magic constants (such as port 6060) in multiple places. Define them in common packages.


## 129. Terminate the Test Process If Namespace Restorals Fail

-   **Fail Fast:** If `setns` fails to restore the init namespace, terminate the process immediately. Continuing without namespace restrictions causes false positives or machine destruction.

## 130. Declare `exitf` (C++) as noreturn Without Unneeded `return`

-   **Executor Hygiene:** If a termination utility halts the executor, do not add unneeded `return` statements after it.

## 131. Unexport Utility Functions Unless Needed Externally

-   **API Conservatism:** Hide traverse functions (such as `ForeachSubArg`) from external packages until they are definitively needed by them.

## 132. Use Capacity 1 Channels for Async Notifications

-   **Concurrency Hygiene:** If a channel is used for notifications, give it a capacity of 1 so the sender does not block if the receiver is busy.

## 133. Avoid Double Locking on Standard Thread-Safe Loggers

-   **Concurrency Idioms:** Standard logging libraries are thread-safe. Do not use custom mutexes to wrap logging calls unless there's a specific reason.

## 134. Do Not Suffix Log Messages with Newlines (`\n`)

-   **Logging Uniformity:** Linter rules prohibit suffixing log messages with a newline character. Stick to plain messages.

## 135. Check for Logical Channel Races if Multiple Groups Close Them

-   **Concurrency Safety:** Do not close channels while concurrent senders might be sending to them. Wait for all senders to finish or use a single sender.


## 136. Keep Stats as Integers Rather than Strings

-   **Performance:** Strings invite maps and mutexes. Dense integer arrays updateable atomically are much faster and generate less garbage.

## 137. Do Not Use `bufio.Writer` When Writing to `bytes.Buffer`

-   **Performance:** A `bytes.Buffer` is already fast in-memory. Wrapping it in `bufio.Writer` only adds unnecessary memory allocations and copies.

## 138. Reduce Variable Scopes Where Possible

-   **Go Idioms:** Declare variables inside `if` statements if they are only needed there (e.g., `if err := ...; err != nil`). This avoids unintentional shadow bugs.

## 139. Use `t.Cleanup` for Shared Test Teardowns

-   **Testing Hygiene:** Use `t.Cleanup` in helper functions (such as `buildExecutor` or `newProc`) to tear down shared states cleanly without cluttering the main test loop.

## 140. Use Buffered Channels for Slow Disk Flush Pipes

-   **Concurrency Hygiene:** If a channel is used to pipe corpus updates to a slow disk flusher, buffer it so the sender does not block on write delays.

## 141. Move Serialization Out of Critical Sections

-   **Performance:** Program serialization or formatting does not need to be under a mutex. Move it out to reduce lock contention.

## 142. Use Standard Scientific Notations for Large Multiples (e.g., `1e6`)

-   **Uniformity:** Instead of counting zeros (e.g., `1000000`), use `1e6` for better readability.


## 143. Print Raw Unparsed Output Traces Using `%q` for Broken Diagnostics

-   **Logging Hygiene:** If a parser fails, do not just print numbers. Print the raw trace with `%q` so that byte escape characters are legible.

## 144. Drop Unneeded Parentheses in Go Expressions

-   **Go Style:** Unlike C++, operators and expressions in Go rarely require heavy parenthesis grouping. Keep them lean!

## 145. Use `Locked` Suffix For Mutex-Guarded Call-Trees

-   **Go Standards:** Standard Go library conventions favor the suffix `Locked` (e.g., `fooLocked()`) over any unexpressed `Unlocked` semantics.

## 146. Avoid Redundant Field Prefixes Inside Struct Declarations

-   **Naming Cleanliness:** If a struct defines `CoverPoint`, do not prefix fields with `cp_`. The context is already clear!

## 147. Use Unique IDs (like PCs) to Map Basic Blocks

-   **Coverage Hygiene:** Using `line:col` can merge distinct basic blocks if multiple branch evaluation expressions are on the same line. Favor unique identifiers like `PC`.


## 148. Use `iota` for Standard Enum and State Definitions

-   **Go Style:** Use `iota` instead of manual integer assignment for states/types.

## 149. Rely on Go's Typeless Fractional Constant Features (e.g., `1/3`)

-   **Precision:** Standard literals like `1/3` can evaluate typeless precision without loss during conversions to float types.

## 150. Group Public Struct Members First Before Private ones

-   **Readability:** Keep public exported fields at the top of struct definitions to make them readable for external consumers.

## 151. Rename Existing Metrics Rather than Creating Duplicates

-   **Hygiene:** If you are slightly tweaking the semantics of a metric, rename the old one rather than adding a confusing duplicate.

## 152. Never Place Returns After C++ `exitf/fail` Utilities

-   **Dead Code:** Utility functions like `fail`, `failmsg`, and `exitf` never return. Any `return` or `break` placed after them is dead code.


## 153. Use Strong Constants (e.g., `targets.Linux`) Instead of Raw Standard Strings

-   **Type Safety:** Avoid comparisons against raw literals like `"linux"`. Use defined constants to prevent silent typos.

## 154. Catch Panics if Parsing Untrusted Data From the VM Client

-   **Robustness:** Data coming from the VM executor (e.g., Flatbuffers or logs) can be corrupted. Go parsers assume well-formed data and might panic. Catch recoveries at the boundary to prevent host manager crashes.

## 155. Propagate Timeout Scaling (e.g., `Scale`) Throughout the System

-   **Testing Hygiene:** If you are running slow emulated architectures (like QEMU with KASAN), they can be 10x slower. Ensure that timeout scaled metrics are propagated everywhere to avoid unexpected timeouts.

## 156. Combine Small Redundant Methods into 1 Unified Simple API

-   **Architecture:** Instead of forcing multiple method calls on a package client, combine them into 1 single-shot API for simplicity and stability.


## 157. Avoid "Just-In-Case" Mutexes and Locking

-   **Concurrency Hygiene:** Locks should only be used when concurrent mutations can happen. Unnecessary mutexes confuse readers and disable race detector capabilities (which help catch un-intended concurrent mutations).

## 158. Drop `else` after `return` or `exitf`

-   **Go Style:** When the primary branch exits, any subsequent evaluation happens in the un-indented main block. This reduces nesting.

## 159. Don't Add Checks Only to Panic if the Standard Crash Trace Would Clear It Up Uniquely

-   **Simplification:** If a null pointer dereference will produce a readable panic trace, do not wrap it with another manual check. Keep code minimal.

## 160. Drop Unneeded Package Renamings in Imports

-   **Go Idioms:** If the package name matches the base of its import path (e.g., `github.com/.../batch`), do not rename it (`batch "github.com/.../batch"`).

## 161. Do Not Use `()` to Refer to Functions in Comments

-   **Style:** Go comments represent names as plain text. Functions are not special (variables, arguments, and structs are also names).

## 162. Rely on Standard Interfaces (like `io.Closer`) Over Custom Ones

-   **Architecture:** When a type has clean-up requirements, let it implement `io.Closer` instead of a custom `Close()` method to make it interoperable.

## 163. Use Dense Indices `[0, Count)` for Rebootable Entities (and VMs)

-   **Simplicity:** Avoid giving VMs new IDs after reboot. Keep them within a dense range of fixed indices to simplify tracking without dynamic mapping structures.

## 164. Use `[size[N]]` in Syzlang to Pad Unions of Commands Automatically

-   **Safety:** Padding union types automatically ensures that if one command changes size, it doesn't break alignment for others.


## 165. Log Client Errors for Authorized Clients (Our Own Bots)

-   **Logging Hygiene:** If a client request fails due to bad input and the client is our own bot/agent, log it as an error (not a user bad request). We need to know if our own tools are buggy.

## 166. Validate String Enum Values Against Allowed Lists

-   **Robustness:** If you use a string field for enums, verify that the value is actually one of the allowed values. Constant typos are very common.

## 167. Use `filepath.Join` for OS-Agnostic Path Concatenation

-   **Uniformity:** Do not use `/` manually. `filepath.Join` properly handles duplicate/missing separators and knows how to use `\` if running on Windows.

## 168. Never Use Arbitrary Strings as Formatters

-   **Security/Type Safety:** Passing an arbitrary string to a formatting function (like `fmt.Printf` or `Failf`) as the first argument is unsafe because the string can contain `%`. Always use `%s` context.

## 169. Inline Short Helper Functions for Reproducers

-   **Reproducer Hygiene:** All code appears in reproducers, so we balance between code structuring and verbosity. Shorter reproducers are easier to read for humans.

## 170. Fail Fast at the Top of the Loop

-   **Robustness:** Fuzzer can well violate your min length requirements. Check constraints at the start of loop bodies before executing side-effects.


## 171. Use Autogenerated Notices for Machine-Generated Files

-   **Uniformity:** Prefix files with `# Code generated by [Tool]. DO NOT EDIT.` to prevent manual edits that get overwritten.

## 172. Make `Invalid` the First Value in Enums (0)

-   **Robustness:** This catches uninitialized fields (Go defaults to zero) and helps identify bugs with uninitialized types.

## 173. Unexport Internal Types and Constants

-   **Encapsulation:** If a type or constant is not needed outside its package, unexport it (make it lowercase) to keep the API surface minimal.

## 174. Avoid Redundant Checks for Empty Slices Before Loops

-   **Style:** `for range` handles empty slices naturally without an explicit check. Keep the code minimal.

## 175. Isolate Complex Closures in `slices.CompactFunc` to Separate Variables or Functions

-   **Style:** Doing it inline makes loop expressions unreadable. Assign to a variable just before the call.

## 176. Avoid `panic` Mount Options in Fuzzer Descriptors

-   **Robustness:** Fuzzer provides invalid images; kernel panics are expected but shouldn't be treated as syzkaller bugs if they were explicitly requested via options. Filter them out.


## 177. Pass Immutable Strings by Const Reference or `std::string_view` in C++

-   **C++ Style/Performance:** To avoid unnecessary memory allocations and data copies, pass strings by `const std::string&` or `std::string_view`.

## 178. Rely on Command Exit Status Rather Than Brittle Error Substrings

-   **Robustness:** Substrings change across versions (e.g., `git` errors). Use numeric exit codes or standard methods to check command success.

## 179. Explain *Why* You are Doing Something, Not Just *What*

-   **Documentation Hygiene:** If you add a filename suffix to avoid ambiguity, document that it's for disambiguation among identically named policies.

## 180. Verify Subprocess Results by Simulating Syntax Errors

-   **Testing Robustness:** Test your error reporting by intentionally breaking inputs (like introducing syntax errors in test files) to see what your tool prints.

## 181. Don't Log Variable Names to End Users

-   **Logging Hygiene:** Display natural language traces instead of internal debugging variable names (e.g., prefer "discovered N source files..." over "discovered len(allUnits) source files...").


## 182. Use `timeNow` Over `time.Now` for Test Stubbing in Dashboard

-   **Go Style/Testing:** Using `timeNow` allows deterministic time testing and prevents flakiness.

## 183. Set Timeouts to 2-5x of Observed Execution Time

-   **Testing Hygiene:** Dampens flakiness without severe penalties. There is no downside to setting it to a higher value (besides avoiding future false positives).

## 184. Use `git push` Directly to URLs to Avoid Adding Remotes

-   **Uniformity:** If you are pushing a specific commit to a remote repo, use `git push [repo_url] [commit]` instead of creating a local remote name.

## 185. Use Standard `auto_todo` Stubs for Unhandled Conversions

-   **Type Safety:** Instead of using a fake `intptr` (or returning empty strings), use a custom type name (like `type auto_todo intptr`) that flags limitations in generated code.

## 186. Use `docker build --no-cache` for Fresh Builds

-   **Docker Hygiene:** Don't break `apt-get` caching rules in Dockerfiles just to get updates. Let the engine's `--no-cache` flag handle it.


## 187. Validate Manipulated Programs Before Persistence

-   **Robustness:** If you modify programs (remove calls, etc.), ensure the resulting program is serialized and validated before saving it to the DB to avoid dangling resources.

## 188. Use `targets` Constants Over Raw Strings in Tests

-   **Type Safety:** Instead of using `"linux"` and `"amd64"`, use `targets.Linux` and `targets.AMD64` from the `pkg/targets` package.

## 189. Use Raw String Literals for Multiline Test Data

-   **Go Style:** When defining multiline test programs, use backtick raw string literals instead of array concatenation or manual joining.

## 190. Prefer `t.Fatal(err)` Over `t.Fatalf("%v", err)`

-   **Go Style:** For simple error reporting where no formatting is needed, use plain `t.Fatal` to keep trace minimal.

## 191. Use Standard `testify/assert` (or `require`) for Test Assertions

-   **Testing Hygiene:** Prefer `assert.Equal(t, want, got)` from `github.com/stretchr/testify/assert` rather than manual `if` comparisons.


## 192. Keep Custom Data Operations in the Owner Package

-   **Architecture:** If an operation does non-trivial program manipulation (like validation), it should live in the `prog` package, rather than in some leaf tool.

## 193. Use Standard `godoc` Formatting for Functions

-   **Documentation Hygiene:** The canonical function comment format understood by godoc/etc is: `// FunctionName does ...`

## 194. Unexport Internal Struct Fields

-   **Encapsulation:** If a field is only read/written within its own package, unexport it (make it lowercase) to keep the API surface minimal.

## 195. Prefer 2-Space Indentation for Text Reports and Logs

-   **Logging Hygiene:** Using 1 space for indentation looks like a typo. Use 2 spaces for visual distinctness.

## 196. Avoid "Syzbot" Prefixes in Dashboard Code

-   **Uniformity:** Use descriptive names of what the component does rather than the infrastructure name (e.g., prefer `db_export` over `syzbot_db_export`).


## 197. Use Hardcoded Reports for Template Tests

-   **Testing Hygiene:** When testing email/text templates, hardcode a real report example to verify line lengths, visual layout, and formatting.

## 198. Don't Test Configurations That Never Run in Production

-   **Testing Practice:** Test the actual production configurations (e.g., varying triggering probability) rather than artificial setups that never occur in real life.

## 199. Keep Email Prefixes Short

-   **Formating:** Subject prefixes appear in limited width UI spaces (dashboards, email clients). Keep them concise (e.g., use `rust:` instead of `rust_kernel:`).

## 200. Validate Prefix Subsets for Ambiguity

-   **Logic Safety:** Ensure that a new prefix isn't a subset of an existing one if it causes shadowing (e.g., don't have `KASAN:` and then `KASAN: use-after-free` without proper ordering).

## 201. Use Wrapper Functions for Uniform Error Policies

-   **Architecture:** Consolidate error handling and logging logic by wrapping handlers, avoiding duplicating `log.Errorf` or `http.Error` on every path.


## 202. State the Reason for a Broken Compiler

-   **Go Style/Debugging:** State why the compiler is broken in `BrokenCompiler` (e.g., "cant-build-arm64-on-windows"), rather than just setting the binary name to a garbage string.

## 203. Use Raw String Literals for Multi-Line C++ Assembly Blocks

-   **C++ Style:** When defining multiline C++ `asm` blocks, use `R"()"` raw string literals instead of array concatenation or manual joining to avoid clutter with `\n\t`.

## 204. Strip Unused Shared Headers from Reproducers

-   **Code Stripping Hygiene:** Wrap system-specific headers (like `kvm.h`) in ifdefs so they can be stripped automatically when the reproducer doesn't require them.

## 205. Avoid Excessive Redundant Safety Checks

-   **Simplicity Principle:** High-quality code strikes a balance between safety and complexity. If you have static checks (lint, etc.) and CI tests for an unlikely mistake, do not add runtime dynamic checks if it increases code complexity.

## 206. Be Mindful of Regex Over-Greediness in Code Generation

-   **Code Gen Hygiene:** Simple regex patterns like `.*` can be over-greedy and wipe out unrelated content. Use non-greedy variants or more specific patterns to avoid future trouble.


## 207. Don't Shell Out if a Go Package Exists

-   **Go Style/Reliability:** Use standard Go libraries (e.g., `oauth2/google`) instead of calling external tools (e.g., `gcloud`) to get OAuth tokens. It's more reliable and performant.

## 208. Use Raw String Literals for Prompt Hardcoding

-   **Prompt Engineering:** When embedding LLM prompts in Go code, use backtick raw string literals to avoid quote escaping clutter. It makes prompt editing much more readable.

## 209. Be Precise with Transient Error Retries (Check Substrings)

-   **Robustness:** Generic errors like `503 Service Unavailable` can be returned for many reasons. If you only want to retry on overload, check for specific substrings (e.g. "The model is overloaded").


## 210. Move Helper Functions to the Bottom of the Test File

-   **Go Style/Readability:** Keep helper functions (like `tickRandom`) at the bottom of the test file to keep the main test logic prominent at the top.

## 211. Add Motivation Comments for Non-Obvious Decisions

-   **Documentation Practice:** Always explain the *why* behind unusual configurations or choices (e.g., limits, sleeps). Don't just state what the code does.

## 212. Prefer Shorter Retries with High Frequencies Over Long Sleeps

-   **Testing Hygiene:** If you are waiting for a subprocess or a network boot, shorter retries with higher frequency are generally better than longer sleeps. It yields the same total time but responds much faster.

## 213. Use Tabs for HTML Templates

-   **Formatting:** Maintain project consistency by using tab characters for indentation in `.html` templates.


## 214. Avoid Tool Slice Aliasing with Cloning

-   **Go Style/Concurrency:** Use `slices.Clone` when appending tools to standard lists in `aflow` workflows. This prevents mutations that affect other workflows running concurrently.

## 215. Hard Errors for Workflow Bugs (Unfixable by LLMs)

-   **Architecture:** If a tool fails due to a bug in our code (e.g., failed to load a target, empty file), it should be a hard error (workflow failure), not returned as a soft error to the LLM agent. The LLM can't fix it.

## 216. Avoid Type Checking in Generic Orchestrators

-   **Architecture/Decoupling:** Don't do `if typ == WorkflowRepro` in a generic handler. Pass extra arguments via a generic map to keep things decoupled.

## 217. Avoid Redundant Prefixes in Tool/Package Names

-   **Naming Practice:** "Tool" or "Syz" parts in names are often excessive. If you are in a `tool` package, things are tools. If you are in `syzkaller`, things relate to `syz`. Keep it short.


## 218. Avoid Generic "Utils" Packages (Junk Drawer)

-   **Architecture/Go Standards:** Do not create or use a global `pkg/util` for domain-agnostic helpers. Keep utilities in their specific contextual packages (e.g., `pkg/email`, `pkg/report`) to prevent unorganized junk drawers.

## 219. Use Single-Line Test Checks (e.g., `require`)

-   **Testing Style:** Instead of multi-line `if err != nil { t.Fatal(err) }`, use standard libraries like `stretchr/testify/require` to check conditions in one line: `require.NoError(t, err)`.

## 220. Propagate Temp File Creation Errors

-   **Robustness:** Silently ignoring errors when creating temporary files can cause difficult-to-track bugs. Propagate them or log them loudly so they are visible in monitoring.

## 221. Use Raw String Literals for Multi-line Test Inputs/Outputs

-   **Testing Practice:** When hardcoding multi-line inputs in tests, use raw string literals. This makes it clear what spaces, tabs, and newlines are tested without the clutter of string concatenation.

> [!TIP]
> This file is automatically updated by the workflow in `.github/workflows/update-review-guidelines.yml` to reflect new feedback as it occurs!
