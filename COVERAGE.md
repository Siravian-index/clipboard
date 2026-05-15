# Test Coverage Report

Generated: 2026-05-15

**Total coverage: 17.1%**

---

## Summary by Package

| Package | Coverage | Status | Notes |
|---|---|---|---|
| `watcher` | 79.3% | ✅ Good | Core polling logic well covered |
| `config` | 79.2% | ✅ Good | Load/Save paths covered |
| `history` | 52.0% | ⚠️ Partial | New methods (Search, Count) uncovered |
| `daemon` | 35.2% | ⚠️ Low | handleConn covered, lifecycle helpers not |
| `ui` | 1.7% | ❌ Minimal | Fyne requires display — only scaleContain tested |
| `main` | 0.0% | ❌ None | Entry point, no unit tests |
| `client` | 0.0% | ❌ None | Requires live Unix socket |
| `hotkey` | 0.0% | ❌ None | Requires X11 display |

---

## Detail by Function

### `config`
| Function | Coverage |
|---|---|
| `Default` | 100.0% |
| `configPath` | 75.0% |
| `Load` | 83.3% |
| `Save` | 71.4% |

### `daemon`
| Function | Coverage |
|---|---|
| `broadcast` | 100.0% |
| `subscribe` | 100.0% |
| `unsubscribe` | 100.0% |
| `handleConn` | 78.9% |
| `broadcastRefresh` | 0.0% |
| `reloadConfig` | 0.0% |
| `maxEntries` | 0.0% |
| `NewServer` | 0.0% |
| `Run` | 0.0% |
| `dataDir` / `ensureDataPaths` / `pidFile*` / `ReadPID` | 0.0% |

### `history`
| Function | Coverage |
|---|---|
| `MemoryHistory.Add` | 100.0% |
| `MemoryHistory.List` | 100.0% |
| `MemoryHistory.Clear` | 100.0% |
| `NewMemoryHistory` | 100.0% |
| `SQLiteHistory.List` | 82.4% |
| `SQLiteHistory.SetMaxSize` | 100.0% |
| `SQLiteHistory.Close` | 100.0% |
| `SQLiteHistory.Add` | 66.7% |
| `SQLiteHistory.Clear` | 63.6% |
| `SQLiteHistory.NewSQLiteHistory` | 66.7% |
| `MemoryHistory.Search` | 0.0% |
| `MemoryHistory.Count` | 0.0% |
| `SQLiteHistory.Search` | 0.0% |
| `SQLiteHistory.Count` | 0.0% |
| `SQLiteHistory.MaxSize` | 0.0% |
| `SQLiteHistory.ImageDir` | 0.0% |
| `EnsureImageDir` | 0.0% |

### `watcher`
| Function | Coverage |
|---|---|
| `NewPollingWatcher` | 100.0% |
| `Start` | 90.0% |
| `poll` | 84.4% |
| `Stop` | 100.0% |
| `sha256Hash` | 100.0% |
| `saveImage` | 66.7% |
| `Reset` | 0.0% |

### `ui`
| Function | Coverage |
|---|---|
| `scaleContain` | 100.0% |
| Everything else | 0.0% |

### `client` / `hotkey` / `main`
All functions: 0.0% — require live runtime environment (Unix socket, X11 display).

---

## Coverage Gaps — Priority Order

### High Priority (testable, critical logic)
- `history`: `Search`, `Count`, `MaxSize` — new methods added without tests
- `daemon`: `broadcastRefresh`, `reloadConfig`, `maxEntries` — live config reload logic

### Medium Priority (testable with mocks)
- `daemon`: `NewServer`, filesystem helpers (`dataDir`, `ensureDataPaths`, `pidFile*`)
- `watcher`: `Reset`, `saveImage` error paths

### Low Priority (require runtime environment)
- `ui`: Fyne widgets require a display — integration tests or `fyne.test` driver
- `client`: requires live Unix socket connection to daemon
- `hotkey`: requires X11 display

---

## Why Total Coverage is Low

The 17.1% total is heavily pulled down by `ui` and `client` which together represent a large portion of the codebase but are inherently difficult to unit test without a graphical environment. The packages that **can** be unit tested (`config`, `watcher`, `history`) sit at 52–79%, which reflects reasonable coverage for the testable surface area.
