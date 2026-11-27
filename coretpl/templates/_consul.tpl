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


{{/*
The namespace of the Consul installation, for spec.operatorNamespace of the ConsulACL and ConsulKV CRs.
The ACL operator runs next to Consul, so the namespace is read from CONSUL_URL,
http://<consul>.<namespace>[.<domain>]:<port>. The render fails when the second label of the
host is missing, empty, or all digits, as in an IP address, since none of those names a namespace.
*/}}
{{- define "coretpl.consulacl.operatorNamespace" -}}
{{- $host := first (splitList ":" (last (splitList "://" (.Values.CONSUL_URL | default "" | toString)))) -}}
{{- $parts := splitList "." $host -}}
{{- $namespace := "" -}}
{{- if ge (len $parts) 2 -}}
{{- $namespace = index $parts 1 -}}
{{- end -}}
{{- if or (eq $namespace "") (regexMatch "^[0-9]+$" $namespace) -}}
{{- fail (printf "CONSUL_URL %q names no namespace. Set it to http://<consul>.<namespace>:<port>." (.Values.CONSUL_URL | default "" | toString)) -}}
{{- end -}}
{{- $namespace -}}
{{- end -}}

{{- define "coretpl.consulacl.enabled" -}}
{{- $mode := include "coretpl.consulAuthMode" . -}}
{{- if and (.Capabilities.APIVersions.Has "netcracker.com/v1/ConsulACL") (has $mode (list "hybrid" "k8s")) -}}
true
{{- end -}}
{{- end -}}
