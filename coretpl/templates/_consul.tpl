{{/*
Consul auth mode, read as the services read it: CONSUL_AUTH_MODE as is; when it is empty, M2M_AUTH_MODE matched
case-insensitively with surrounding whitespace ignored; legacy when both are empty.
Fails the render on any value other than legacy, hybrid, or k8s.
*/}}
{{- define "coretpl.consulAuthMode" -}}
{{- $mode := .Values.CONSUL_AUTH_MODE | default "" | toString -}}
{{- if not $mode -}}
{{- $mode = .Values.M2M_AUTH_MODE | default "" | toString | trim | lower | default "legacy" -}}
{{- end -}}
{{- if not (has $mode (list "legacy" "hybrid" "k8s")) -}}
{{- fail (printf "Unknown Consul auth mode %q from CONSUL_AUTH_MODE or M2M_AUTH_MODE: expected legacy, hybrid, or k8s" $mode) -}}
{{- end -}}
{{- $mode -}}
{{- end -}}


{{- define "consulacl.operatorNamespace" -}}
{{- index (splitList "." (first (splitList ":" (last (splitList "://" .Values.CONSUL_URL))))) 1 -}}
{{- end -}}

{{- define "consulacl.enabled" -}}
{{- $mode := include "coretpl.consulAuthMode" . -}}
{{- if and (.Capabilities.APIVersions.Has "netcracker.com/v1/ConsulACL") (has $mode (list "hybrid" "k8s")) -}}
true
{{- end -}}
{{- end -}}
