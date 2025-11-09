## Appendix F. Bash Behavior (Informative)

The following observations are based on minimal reproductions on Bash (environment: `bash -n` and execution). They motivate the places where the EnvSeed parser aligns with Bash syntax. This appendix is informative; normative requirements remain in Sections 4 and 5.

- Top-level trailing comments
  - Input:
    ```sh
    VAR=one # comment
    printf '%s\n' "$VAR"
    ```
    - `bash -n`: succeeds / runtime output: `one`
  - A `#` inside double quotes does not start a comment:
    ```sh
    VAR="a # b"; printf '%s\n' "$VAR"
    ```
    - `bash -n`: succeeds / runtime output: `a # b`
  - A `#` occurring as part of a word inside command substitution does not start a comment:
    ```sh
    VAR=$(echo hi#there); printf '%s\n' "$VAR"
    ```
    - `bash -n`: succeeds / runtime output: `hi#there`
  - The same holds inside backticks:
    ```sh
    VAR=`printf '%s' hi#there`; printf '%s\n' "$VAR"
    ```
    - `bash -n`: succeeds / runtime output: `hi#there`
