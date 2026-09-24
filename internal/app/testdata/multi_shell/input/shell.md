---
uchi: v1
title: Multi-shell Configuration
---

# Common Aliases

```bash {schema=alias}
alias g="git"
```

# Bash-only Config

```bash {schema=alias target=bash}
alias b="bash_specific"
```

# Zsh-only Config

```zsh {schema=alias target=zsh}
alias z="zsh_specific"
```

# Shared Bash and Zsh Config

```sh {schema=alias target="bash,zsh"}
alias bz="shared_bash_zsh"
```
