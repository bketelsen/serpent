{{- /* Heavily inspired by the Go toolchain and fd */ -}}
{{ with .Short }}
{{- .  | wrapTTY }}
{{"\n"}}
{{- end}}
{{prettyHeader "Usage"}}
{{indent .FullUsage 2}}
{{- with .Deprecated }}
{{- indent (printf "DEPRECATED: %s" .) 2 | wrapTTY }}
{{"\n"}}
{{- end }}

{{ with .Aliases }}
{{"  Aliases: "}} {{- joinStrings .}}
{{- end }}

{{- with .Long}}
{{"\n"}}
{{- indent . 2}}
{{ "\n" }}
{{- end }}
{{ with visibleChildren . }}
{{- range $index, $child := . }}
{{- if eq $index 0 }}
{{ prettyHeader "Subcommands"}}
{{- end }}
    {{- "\n" }}
    {{- formatSubcommand . | trimNewline }}
{{- end }}
{{- "\n" }}
{{- end }}
{{- range $index, $group := optionGroups . }}
{{ with $group.Name }} {{- print $group.Name " Options" | prettyHeader }} {{ else -}} {{ prettyHeader "Options"}}{{- end -}}
{{- with $group.Description }}
{{ formatGroupDescription . }}
{{- else }}
{{- end }}
    {{- range $index, $option := $group.Options }}
	{{- if not (eq $option.FlagShorthand "") }}{{- print "\n "}} {{ keyword "-"}}{{keyword $option.FlagShorthand }}{{", "}}
	{{- else }}{{- print "\n      " -}}
	{{- end }}
    {{- with flagName $option }}{{keyword "--"}}{{ keyword . }}{{ end }} {{- with typeHelper $option }} {{ . }}{{ end }}
    {{- with envName $option }}, {{ print "$" . | keyword }}{{ end }}
    {{- with $option.Default }} (default: {{ . }}){{ end }}
        {{- with $option.Description }}
            {{- $desc := $option.Description }}
{{ indent $desc 10 }}
{{- if isDeprecated $option }}{{ indent (printf "DEPRECATED: Use %s instead." (useInstead $option)) 10 }}{{ end }}
        {{- end -}}
    {{- end }}
{{- end }}
{{- if .Parent }}
———
Run `{{ rootCommandName . }} --help` for a list of global options.
{{- else }}
{{- if .ContactInfo }}
{{ prettyHeader "Contact"}}
{{- with .ContactInfo }}
{{- with .RepoLink }}{{- print "\n "}}Repository:
    {{ keyword . }}{{ end }}
{{- with .IssuesLink }}{{- print "\n "}}Issues:
    {{ keyword . }}{{ end }}
{{- with .ChatLink }}{{- print "\n "}}Chat:
    {{ keyword . }}{{ end }}
{{- with .EmailLink }}{{- print "\n "}}Email:
    {{ keyword . }}{{ end }}
{{- end }}
{{- else }}
{{- end }}
{{- end }}

