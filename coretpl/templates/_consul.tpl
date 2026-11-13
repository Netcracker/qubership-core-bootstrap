{{/*
Consul auth mode: CONSUL_AUTH_MODE, else M2M_AUTH_MODE, else legacy.
Fails the render on any value other than legacy, hybrid, or k8s.
*/}}
{{- define "coretpl.consulAuthMode" -}}
{{- $mode := .Values.CONSUL_AUTH_MODE | default .Values.M2M_AUTH_MODE | default "legacy" | trim -}}
{{- if not (has $mode (list "legacy" "hybrid" "k8s")) -}}
{{- fail (printf "Unknown Consul auth mode %q from CONSUL_AUTH_MODE or M2M_AUTH_MODE: expected legacy, hybrid, or k8s" $mode) -}}
{{- end -}}
{{- $mode -}}
{{- end -}}


{{- define "consulacl.operatorNamespace" -}}
{{- index (splitList "." (first (splitList ":" (last (splitList "://" .Values.CONSUL_URL))))) 1 -}}
{{- end -}}

{{- define "consulacl.enabled" -}}
{{- $mode := .Values.CONSUL_AUTH_MODE | default .Values.M2M_AUTH_MODE | default "legacy" | toString -}}
{{- if and (.Capabilities.APIVersions.Has "netcracker.com/v1/ConsulACL") (ne $mode "legacy") -}}
true
{{- end -}}
{{- end -}}
