# bash completion for opencode-sm
# Install: copy to /etc/bash_completion.d/ or ~/.local/share/bash-completion/completions/

_opencode_sm() {
    local cur prev opts
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"

    opts="--db-path --watch --version --help"

    if [[ ${cur} == -* ]] ; then
        COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
        return 0
    fi

    case "${prev}" in
        --db-path)
            COMPREPLY=( $(compgen -d -- ${cur}) )
            return 0
            ;;
    esac

    COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
    return 0
}

complete -F _opencode_sm opencode-sm
