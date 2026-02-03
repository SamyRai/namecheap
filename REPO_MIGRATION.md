# Repository strategy: keep Namecheap vs. switch to multi-provider (ZoneKit)

**Summary**

Two viable paths:

1. **Keep this repo as Namecheap-only** (maintenance-only here) and create a **new repository** for the multi-provider implementation.
2. **Rename and rebrand this repo** (e.g., `zonekit`), update module path and docs, and continue building the provider-agnostic code here.

Recommendation: Given the current state (large refactor already present on the feature branch and low adoption/stars), the simplest path is to **rename/rebrand this repository to `zonekit` and continue here**, provided you are comfortable with a breaking change / major-version bump and will publish migration guidance. If *backwards compatibility* is critical for existing users, prefer Option 1 (split into a new `zonekit` repository and keep this repo as `namecheap` maintenance).

---

## Findings

- There is 1 open PR: `feature/generic-dns-provider-plugin` (3 commits, adds provider-agnostic packages, adds OpenAPI providers, rebranding touches like `.gitmodules`, docs and `go.mod` module set to `zonekit`). ✅
- Local branch is up to date with remote; branch is 3 commits ahead of `master` and contains the multi-provider work. ✅
- Repo adoption is small (stars/watchers low), so breaking changes are low-risk for wide users. ✅

---

## Option A — Keep this repo as Namecheap-only (create a new repo for ZoneKit)

Pros:

- Preserves stable public interface and name for existing users.
- Clear separation of responsibility: this repo becomes a maintenance-only place for Namecheap-specific functionality.
- Less risk of confusing users who expect Namecheap-only tooling.

Cons:

- Extra overhead to maintain two repositories.
- Requires copying or preserving multi-provider work in a new repo while retaining history.

Steps (high level):

1. Create new repo `zonekit`.
2. From this local repo, push the feature branch into the new repo preserving history:
   - `git remote add zonekit git@github.com:SamyRai/zonekit.git`
   - `git push zonekit feature/generic-dns-provider-plugin:master`
3. In this repo, revert/clean the feature commits (or create a `maintenance` branch that excludes them):
   - Use `git revert` for the 3 commits or reset master to the previous commit and push a `maintenance` branch for the Namecheap-focused work.
4. Update `README.md` in both repos with cross-links and migration notes.
5. Tag and release a maintenance version for `namecheap` and create a new initial release for `zonekit` (major version as needed).
6. Add a deprecation/migration policy and timeline in `RELEASES.md` (e.g., 3–6 months grace period).

Risk mitigation:

- Keep a `legacy` branch with the current Namecheap behavior and tests.
- Publish migration steps and add automated tests in `zonekit` that verify Namecheap adapter still works.

---

## Option B — Rename repo and continue multi-provider here (recommended if few users rely on current API)

Pros:

- Single source of truth; no repository split overhead.
- The feature branch is already in this repository (minimal friction to adopt the refactor).
- Keeps commit history intact and reduces complexity.

Cons:

- Breaking change: module path and imports will change (requires major version bump and coordination with users).
- Need to update CI, badges, modules, README, docs and possibly change package names.

Steps (high level):

1. Rename the GitHub repository (Settings → Rename) to `zonekit` or another chosen name.
2. Update `go.mod` module to a canonical import path: e.g. `module github.com/SamyRai/zonekit`, run `go mod tidy`.
3. Merge or fast-forward the `feature/generic-dns-provider-plugin` branch into `master` once tests and CI pass.
4. Update all docs, `README.md`, release notes and project description to reflect the new scope.
5. Publish a **major release** (v2.0.0 or v1.0.0 if new) and create a clear migration guide from `namecheap` → `zonekit`.
6. Keep a `legacy` branch or tag for the Namecheap-only code and maintain it for critical fixes for a defined period.

Risk mitigation:

- Add clear migration docs and scripts to help users migrate imports.
- Keep CI to run tests for Namecheap adapter using fixtures.
- Announce the change across README, GitHub release notes, and the issue tracker.

---

## Concrete commands — split repo (Option A)

- Create new repo and push branch as master:

```bash
# create repo on GitHub (web or gh cli)
# locally:
git remote add zonekit git@github.com:SamyRai/zonekit.git
git push zonekit feature/generic-dns-provider-plugin:master
```

- In current repo: revert or create maintenance branch:

```bash
# make a maintenance branch that remains Namecheap-only
git checkout master
git branch maintenance
# reset master to the commit before the provider refactors (use commit SHA):
git reset --hard <sha-before-feature>
git push --force origin master  # be careful: only if you intend to rewrite remote history
git push origin maintenance
```

(Alternative: use `git revert` for the 3 refactor commits instead of history rewrite.)

---

## Concrete commands — rename repo (Option B)

1. Rename repository in GitHub UI (or via API), then locally:

```bash
# update origin URL if your repo URL changed
git remote set-url origin git@github.com:SamyRai/zonekit.git
# update module path
# edit go.mod: module github.com/SamyRai/zonekit
go mod tidy
# run tests and linters
make test || go test ./...
# merge feature branch after validation
git checkout master
git merge origin/feature/generic-dns-provider-plugin
git push origin master
```

1. Publish a major release and create migration instructions for users (how to update imports and commands).

---

## Recommendation & next steps (short)

- If you want to move forward quickly with multi-provider (and are OK with a breaking change), **rename** the repo to `zonekit` and continue (Option B). This is faster and keeps the refactor in place.
- If you must preserve the existing Namecheap public API without breaking existing users, then **split**: create a new `zonekit` repo and make this one maintenance-only (Option A).

Suggested immediate actions (pick one):

1. If renaming: test the feature branch fully, update `go.mod` to `github.com/SamyRai/zonekit`, run CI locally and merge; then rename repo on GitHub and publish a major release.
2. If splitting: create the `zonekit` repo and push the feature branch there (preserve history), then revert the feature commits here and create a `maintenance` branch for Namecheap.

---

If you want, I can:

- create the `REPO_MIGRATION.md` file (done),
- open PRs with the minimal revert (if you choose Option A),
- or prepare a `go.mod` / README / release checklist for Option B so you can rename and release safely.

Tell me which option you prefer and I will prepare the next PR or list of commands to run.
