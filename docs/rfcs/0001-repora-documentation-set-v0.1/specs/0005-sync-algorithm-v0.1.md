# SPEC-0005: Sync Algorithm (v0.1)

Status: Draft

1. Ensure the local bare mirror exists for the repository identity.
2. Configure or update the local remotes for canonical and mirror targets.
3. Fetch canonical first with pruning.
4. Fetch each mirror with pruning.
5. Resolve the comparison ref deterministically:

   - Repora shall attempt to resolve `refs/remotes/<remote>/HEAD` using
     `git remote set-head <remote> -a`.
   - If remote HEAD is not set or is ambiguous, Repora shall resolve the default
     branch via `git symbolic-ref refs/remotes/<remote>/HEAD` or equivalent.
   - The resolved ref shall be cached for the duration of execution to ensure
     consistency across comparisons.
   - If no deterministic ref can be resolved, Repora must fail with a clear
     execution or configuration error.

6. Compute divergence per mirror using Git history comparison, for example:

   ```text
   git rev-list --left-right --count canonical/HEAD...mirror/HEAD
   ```

7. Interpret the result:

   - `0 0`: equal
   - `N 0`: mirror is behind canonical
   - `0 N`: mirror is ahead of canonical
   - `N M`: canonical and mirror have diverged

8. If divergence is detected:

   - Continue only if `--force` is specified.
   - Otherwise fail with exit code `2`.

9. Make the local bare mirror equivalent to canonical state.
10. Push `--mirror` from the local bare mirror to each target mirror.

## Precision Requirement

`git push --mirror` pushes from the local repository to a target remote. Repora
must not imply or assume direct remote-to-remote transfer. The local bare mirror
is the execution substrate and must be synchronized to canonical before it is
used as the source for mirror updates.
