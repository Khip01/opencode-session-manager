# fish completion for opencode-sm
# Install: copy to ~/.config/fish/completions/

function __opencode_sm_completion
    set -l cmd (commandline -opc)
    set -l current (commandline -ct)

    set -l opts --db-path --watch --version --help

    for option in $opts
        if string match -q -- "--*" $current
            and not string match -q -- "$option*" $current
            echo $option
        end
    end
end

complete -c opencode-sm -f -a "(__opencode_sm_completion)"
