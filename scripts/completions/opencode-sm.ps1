# PowerShell completion for opencode-sm
# Install: add to your PowerShell profile ($PROFILE)

Register-ArgumentCompleter -Native -CommandName 'opencode-sm' -ScriptBlock {
    param($wordToComplete, $commandAst, $cursorPosition)

    $options = @(
        '--db-path',
        '--watch',
        '--version',
        '--help'
    )

    $options | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
        [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterName', $_)
    }
}
